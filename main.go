package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

type ClientHeader struct {
	Host string
	UserAgent string
}

// Real proxy function
func proxy(clientConn net.Conn, serverConn net.Conn)  {
	io.Copy(clientConn, serverConn)
	io.Copy(serverConn, clientConn)
}

// Maybe useless temporarily
func connetServer(conn net.Conn, header *ClientHeader) {
}

// Parse client send data
// e.g.
// CONNECT clients4.google.com:443 HTTP/1.1
// Host: clients4.google.com:443
// Proxy-Connection: keep-alive
// User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Safari/537.36
func parseData(data string) (string, *ClientHeader, error){
	header := new(ClientHeader)

	items := strings.Split(data, "\n")
	if len(items) < 1 {
		return "", header, errors.New("client header is not valid")
	}

	connectLine := items[0]
	connectItems := strings.Split(connectLine, " ")
	if len(connectItems) != 3 || connectItems[0] != "CONNECT" {
		return "", header, errors.New("connect line is not valid")
	}

	serverAddress := connectItems[1]

	for _, item := range items[1:] {
		headerItems := strings.Split(item, " ")
		if len(headerItems) == 2 {
			switch strings.ToLower(headerItems[0]) {
			case "host":
				header.Host = headerItems[1]
			case "user-agent":
				header.UserAgent = headerItems[1]
			}
		}
	}

	return serverAddress, header, nil
}


// We already make connection with client when call this function.
// We read client data and parse it.
// And connect to target server for client.
// Then we go to a loop, serve them.
func handleConn(conn net.Conn)() {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr()
	fmt.Println("wait for client: ", remoteAddr, "'s data")

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("client: ", remoteAddr, " error: ", err)
		return
	}

	data := string(buf[:n])
	fmt.Println("receive data: ", data)
	serverAddress, header, err := parseData(data)
	if err != nil {
		fmt.Println("failed to parse client data: ", err)
		return
	}

	serverConn, err := net.Dial("tcp", serverAddress)
	fmt.Println(serverAddress)
	fmt.Println(serverConn)
	if err != nil {
		fmt.Println("failed to connect target server: ", serverAddress, " error: ", err)
		return
	}
	defer serverConn.Close()

	connetServer(serverConn, header)

	_, writeErr := conn.Write([]byte("HTTP/1.1 200 Connection Established"))
	if writeErr != nil {
		fmt.Println("write to client fail, error: ", writeErr)
		return
	}

	fmt.Println("will proxy for both")
	proxy(conn, serverConn)
}


func main() {
	address := "127.0.0.1:17777"
	sock, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}

	fmt.Println("start listen 17777 to proxy")

	for {
		clientConn, err := sock.Accept()
		if err != nil {
			fmt.Println("accept error: ", err)
		}

		go handleConn(clientConn)
	}
}
