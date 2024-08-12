package main

import (
	"log"
	"net"
	"time"
)

func main() {
	// 目标IP和端口，发送数据到 utun7 的 IP 地址
	remoteAddr := &net.UDPAddr{
		IP:   net.ParseIP("192.168.10.4"),
		Port: 6976, // 可以是任意端口号
	}

	// 创建一个UDP连接
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Fatalf("无法创建UDP连接: %v", err)
	}
	defer conn.Close()

	// 构造并发送数据
	message := []byte("Hello, TUN device!")
	for {
		_, err = conn.Write(message)
		if err != nil {
			log.Fatalf("无法发送数据: %v", err)
		}

		log.Printf("发送数据到 utun7: %s", message)

		time.Sleep(2 * time.Second) // 每2秒发送一次
	}
}
