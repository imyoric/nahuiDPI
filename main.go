package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

var BanList = StringList{
	items: []string{}, // Инициализация среза строк
}

func TidyConnect(conn net.Conn, targetConn net.Conn, logStr string, host string) {
	var isUsingNahuiDpi = !isUsingBanList
	if isUsingBanList && BanList.Contains(host) {
		isUsingNahuiDpi = true
	}

	//Verbose Log
	if isVerbose {
		log.Println(logStr, isUsingNahuiDpi)
	}
	//VLog END

	if isUsingNahuiDpi {
		go func() {
			var o = uploadStartPacketSize
			var buffer []byte

			for {
				if o < uploadPacketSizeLimit {
					o += 1
					buffer = make([]byte, o)
				}

				n, err := conn.Read(buffer)
				if err != nil {
					break
				}

				if n == 0 {
					// No more data to read
					break
				}

				if o%128 == 0 && o != uploadPacketSizeLimit {
					time.Sleep(time.Duration(random(12, 40)) * time.Millisecond)
				}

				//Ssssssl error
				//if len(buffer) >= len(host) {
				//	SearchAndUnTidyHost(buffer, host)
				//}
				//
				////fmt.Println(cap(buffer), len(host), o)
				//SearchAndUnTidyHost(buffer, "http/1.1")

				_, err = targetConn.Write(buffer[:n])
				if err != nil {
					break
				}
			}
		}()

		var o = downloadStartPacketSize
		buffer := make([]byte, o)
		for {
			if o < downloadPacketSizeLimit {
				o++
				buffer = make([]byte, o)
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
	io.Copy(conn, targetConn)
}

func SearchAndUnTidyHost(origin []byte, host string) []byte {
	hostBytes := []byte(host)
	index := bytes.Index(origin, hostBytes)

	toReplace := []byte(strings.ToUpper(host))

	if index != -1 {
		for i := 0; i < len(hostBytes); i++ {
			origin[index+i] = toReplace[i]
		}
	}

	return origin
}

func random(min int, max int) int {
	return rand.Intn(max) + min
}

var isVerbose = false

var isUsingBanList = false
var uploadStartPacketSize = 1
var uploadPacketSizeLimit = 1024

var downloadStartPacketSize = 1
var downloadPacketSizeLimit = 1024

var server_port = 8080

var DNSServer = "8.8.8.8"

func main() {
	p := flag.Int("port", 8080, "Server port")
	isB := flag.Bool("banlist", false, "Using ban list?")
	v := flag.Bool("v", false, "Verbose?")
	socks := flag.Bool("socks", false, "Using socks proxy?")
	//s := flag.Bool("system-proxy", false, "System Proxy?")

	u1 := flag.Int("upload_startpacketsize", 1, "Upload start packet size")
	u2 := flag.Int("upload_packetsizelimit", 1024, "Upload packet size limit")

	d1 := flag.Int("download_startpacketsize", 64, "Download start packet size")
	d2 := flag.Int("download_packetsizelimit", 1024, "Download packet size limit")

	dns := flag.String("dns", "8.8.8.8", "Select DNS Server")

	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	uploadStartPacketSize = *u1
	uploadPacketSizeLimit = *u2

	downloadStartPacketSize = *d1
	downloadPacketSizeLimit = *d2

	isVerbose = *v
	DNSServer = *dns

	server_port = *p

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

	if *socks {
		socksProxy(server_port)
	} else {
		httpProxy(server_port)
	}
}
