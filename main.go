package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
)

type MainConfig struct {
	Address  string `json:"address"`
	User     string `json:"auth_user"`
	Password string `json:"auth_pass"`
	Auth     bool   `json:"auth"`
}

type GoppmClient struct {
	Conns    []net.Conn
	IP       string
	Authed   int   // 0 not authed; 1 authing; 2 authed
	AuthTime int64 // timestamp when send auth data
}

var serveClients map[string]*GoppmClient

// Real proxy function
func proxy(clientConn net.Conn, serverConn net.Conn) {
	go io.Copy(clientConn, serverConn)
	io.Copy(serverConn, clientConn)
}

// We already make connection with client when call this function.
// We read client data and parse it.
// And connect to target server for client.
// Then we start serve them.
func handleConn(conn net.Conn, config *MainConfig) {
	defer conn.Close()

	var goppmClient *GoppmClient
	remoteAddr := conn.RemoteAddr().String()
	if _, ok := serveClients[remoteAddr]; !ok {
		// If I declare goppmClient as `var goppmClient GoppmClient`,
		// I don't need use `new(GoppmClient)` to initialize it
		// which was interesting.
		goppmClient = new(GoppmClient)
		goppmClient.IP = remoteAddr
		goppmClient.Conns = make([]net.Conn, 10)
	} else {
		goppmClient = serveClients[remoteAddr]
	}

	fmt.Println("wait for client: ", remoteAddr, "'s data")

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("client: ", remoteAddr, " error: ", err)
		return
	}

	data := string(buf[:n])
	fmt.Println("receive connect data: ", data)
	serverAddress, err := parseConnData(data)
	if err != nil {
		fmt.Println("failed to parse client data: ", err)
		return
	}

	if err := checkAuth(goppmClient, conn, config); err != nil {
		fmt.Println("failed to auth client: ", err)
		return
	}

	serverConn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		fmt.Println("failed to connect target server: ", serverAddress, " error: ", err)
		return
	}
	defer serverConn.Close()

	_, writeErr := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if writeErr != nil {
		fmt.Println("write to client fail, error: ", writeErr)
		return
	}

	fmt.Println("will proxy for both")
	proxy(conn, serverConn)
}

// Parse client send data
// e.g.
// CONNECT clients4.google.com:443 HTTP/1.1
// Host: clients4.google.com:443
// Proxy-Connection: keep-alive
// User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Safari/537.36
func parseConnData(data string) (string, error) {

	items := strings.Split(data, "\n")
	if len(items) < 1 {
		return "", errors.New("client header is not valid")
	}

	connectLine := items[0]
	connectItems := strings.Split(connectLine, " ")
	if len(connectItems) != 3 || connectItems[0] != "CONNECT" {
		return "", errors.New("connect line is not valid")
	}

	serverAddress := connectItems[1]

	return serverAddress, nil
}

func loadConfig(path string) *MainConfig {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panicln("load config conf failed: ", err)
	}

	config := &MainConfig{}
	err = json.Unmarshal(buf, config)
	if err != nil {
		log.Panicln("decode config file failed:", string(buf), err)
	}

	return config
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "./config.json", "代理配置，默认为 ./config.json")
	flag.Parse()

	config := loadConfig(configFile)
	sock, err := net.Listen("tcp", config.Address)
	if err != nil {
		panic(err)
	}

	fmt.Println("start listen: ", config.Address, " to proxy")

	for {
		clientConn, err := sock.Accept()
		if err != nil {
			fmt.Println("accept error: ", err)
		}

		go handleConn(clientConn, config)
	}
}
