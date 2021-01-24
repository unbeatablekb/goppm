package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

func checkAuth(goppmClient *GoppmClient, conn net.Conn, config *MainConfig) error {
	if config.Auth == false {
		return nil
	}

	needAuth := true
	if goppmClient.Authed == 1 && time.Now().Unix()-goppmClient.AuthTime > 60 {
		needAuth = true
	} else if goppmClient.Authed == 0 {
		needAuth = true
	} else {
		needAuth = false
	}

	if needAuth {
		goppmClient.AuthTime = time.Now().Unix()
		if err := forceClientAuth(conn, config); err != nil {
			return err
		}
	}

	return nil
}

func parseAuthData(data string, credentials string) bool {
	lines := strings.Split(data, "\n")
	if len(lines) < 1 {
		return false
	}

	for _, line := range lines {
		items := strings.SplitN(line, " ", 1)
		if len(items) == 2 {
			if items[0] == "Authorization:" && items[1] == credentials {
				return true
			}
		}
	}

	return false
}

// Do 407 auth
// References: https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Status/407
func forceClientAuth(conn net.Conn, config *MainConfig) error {
	authData := "HTTP/1.1 407 Proxy Authentication Required\r\n" +
		"Proxy-Authenticate: Basic realm=\"Access to my cool goppm\"\r\n\r\n"
	_, err := conn.Write([]byte(authData))
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Second)
	proxyCredentials := config.User + ":" + config.Password
	fmt.Println(proxyCredentials)
	credentials := base64.StdEncoding.EncodeToString([]byte(proxyCredentials))
	fmt.Println(credentials)

	var clientData string
	reader := bufio.NewReader(conn)
	// Here I try to use a grace way to handle read.
	// I didn't pay attention to this problem before.
	// But Tcp was a protocol which was Byte-oriented streams.
	// So user should get meanings from tcp by themselves.
	// References: https://wangbjun.site/2019/coding/golang/golang-tcp-package.html
	for {
		line, err := reader.ReadSlice('\n')
		if err == nil {
			return err
		}
		fmt.Println("data is: ", line)
		time.Sleep(time.Second)
	}

	fmt.Println("data is: ", clientData)
	if parseAuthData(clientData, credentials) {
		return nil
	}
	return errors.New("parse auth failed")
}
