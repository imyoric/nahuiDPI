package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Println("Error reading request:", err)
		return
	}

	// Обработка HTTPS
	if req.Method == http.MethodConnect {
		handleConnect(req, conn)
	} else {
		conn.Close()
		//handleHTTPRequest(req, conn)
	}
}

func handleConnect(req *http.Request, conn net.Conn) {
	host := req.URL.Hostname()

	ipOfRes, err := GetPreferredIP(DNSServer+":53", host)
	if err != nil {
		log.Println("Error resolving IP:", err)
		return
	}

	port, _ := strconv.Atoi(req.URL.Port())

	// Устанавливаем соединение с целевым сервером
	targetConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: ipOfRes, Port: port})
	if err != nil {
		log.Println("Error connect to target:", err)
		conn.Close()
		return
	}

	var isUsingNahuiDpi = !isUsingBanList

	if isUsingBanList && BanList.Contains(host) {
		isUsingNahuiDpi = true
	}

	//Verbose Log
	if isVerbose {
		fmt.Println("Connect -> "+host+" ("+ipOfRes.String()+") nahuidpi? =>", isUsingNahuiDpi)
	}
	//VLog END

	if isUsingNahuiDpi {
		go func() {
			var o = uploadStartPacketSize
			buffer := make([]byte, o)
			for {
				if o < uploadPacketSizeLimit {
					o++
				}

				n, err := conn.Read(buffer)
				if err != nil {
					break
				}

				if n == 0 {
					// No more data to read
					break
				}

				_, err = targetConn.Write(buffer[:n])
				if err != nil {
					break
				}
			}
		}()

		conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		var o = downloadStartPacketSize
		buffer := make([]byte, o)
		for {
			if o < downloadPacketSizeLimit {
				o++
			}

			n, err := targetConn.Read(buffer)
			if err != nil {
				break
			}

			if n == 0 {
				// No more data to read
				break
			}

			_, err = conn.Write(buffer[:n])
			if err != nil {
				break
			}
		}

		targetConn.Close()
		return
	}

	go io.Copy(targetConn, conn)

	defer targetConn.Close()
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	io.Copy(conn, targetConn)
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

var BanList = StringList{
	items: []string{}, // Инициализация среза строк
}

var isVerbose = false

var isUsingBanList = false
var uploadStartPacketSize = 1
var uploadPacketSizeLimit = 4096

var downloadStartPacketSize = 1
var downloadPacketSizeLimit = 4096

var DNSServer = "8.8.8.8"

func main() {
	port := flag.Int("port", 8080, "Server port")
	isB := flag.Bool("banlist", false, "Using ban list?")
	v := flag.Bool("v", false, "Verbose?")
	//s := flag.Bool("system-proxy", false, "System Proxy?")

	u1 := flag.Int("upload_startpacketsize", 1, "Upload start packet size")
	u2 := flag.Int("upload_packetsizelimit", 4096, "Upload packet size limit")

	d1 := flag.Int("download_startpacketsize", 64, "Download start packet size")
	d2 := flag.Int("download_packetsizelimit", 4096, "Download packet size limit")

	dns := flag.String("dns", "8.8.8.8", "Select DNS Server")

	flag.Parse()

	uploadStartPacketSize = *u1
	uploadPacketSizeLimit = *u2

	downloadStartPacketSize = *d1
	downloadPacketSizeLimit = *d2

	isVerbose = *v
	DNSServer = *dns

	if *isB {
		isUsingBanList = true
		file, err := os.Open("banlist.txt")
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			t := scanner.Text()
			BanList.Add(strings.TrimSpace(scanner.Text()))
			fmt.Println("Add to banlist: " + t)
		}

		file.Close()
	}

	//Start server
	log.Println("nahuiDPI proxy started at 0.0.0.0:" + strconv.Itoa(*port))
	log.Println("Please setting https proxy in system")
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
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
		go handleConnection(conn)
	}
}
