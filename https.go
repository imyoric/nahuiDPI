package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
)

func handleHTTPSConnection(conn net.Conn) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Println("[HTTPS] Error reading request:", err)
		return
	}

	// Обработка HTTPS
	if req.Method == http.MethodConnect {
		handleHTTPSConnect(req, conn)
	} else {
		conn.Close()
		//handleHTTPRequest(req, conn)
	}
}

func handleHTTPSConnect(req *http.Request, conn net.Conn) {
	host := req.URL.Hostname()

	remoteAddr := conn.RemoteAddr().String()

	ipOfRes, err := GetPreferredIP(DNSServer+":53", host)
	if err != nil {
		if isVerbose {
			log.Println("[HTTPS] Error: client "+remoteAddr+", error resolving ip:", err)
		}
		return
	}

	port, _ := strconv.Atoi(req.URL.Port())

	// Устанавливаем соединение с целевым сервером
	targetConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: ipOfRes, Port: port})
	if err != nil {
		if isVerbose {
			log.Println("[HTTPS] Error: client "+remoteAddr+", error connect to target:", err)
		}
		return
	}

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	TidyConnect(conn, targetConn, "[HTTPS] Connect: client "+remoteAddr+" -> "+host+" ("+ipOfRes.String()+") nahuiDPI? =>", host)
}

func handleHTTPRequest(req *http.Request, conn net.Conn) {
	client := &http.Client{}

	// Установка URI и Host
	req.RequestURI = ""
	req.Host = req.URL.Host

	// Выполнение запроса
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error forwarding request:", err)
		return
	}
	defer resp.Body.Close()

	// Копирование заголовков ответа
	for key, values := range resp.Header {
		for _, value := range values {
			conn.Write([]byte(key + ": " + value + "\r\n"))
		}
	}
	conn.Write([]byte("\r\n")) // Конец заголовков

	// Копирование тела ответа
	io.Copy(conn, resp.Body)
}

func httpProxy(port int) {
	//Start server
	log.Println("nahuiDPI https proxy started at 0.0.0.0:" + strconv.Itoa(port))
	log.Println("Please setting https proxy in system")
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	//Working server
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleHTTPSConnection(conn)
	}
}
