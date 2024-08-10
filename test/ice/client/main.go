package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
	"github.com/pion/ice/v2"
)

const (
	// 硬编码的 ICE 连接的用户名和密码
	hardcodedUfrag = "userfrag"
	hardcodedPwd   = "password"
	stunServer     = "stun:stun.l.google.com:19302"
)

func main() {
	serverAddr := flag.String("addr", "ws://localhost:8080/ws", "signaling server address")
	flag.Parse()

	conn, _, err := websocket.DefaultDialer.Dial(*serverAddr, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	// 创建 ICE agent，并使用 STUN 服务器配置
	agent, err := ice.NewAgent(&ice.AgentConfig{
		NetworkTypes: []ice.NetworkType{ice.NetworkTypeUDP4, ice.NetworkTypeUDP6},
		Urls: []*ice.URL{
			{
				Scheme: ice.SchemeTypeSTUN,
				Host:   stunServer,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 处理 ICE 候选者
	agent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			fmt.Println("Discovered new candidate:", c.String())
			err := conn.WriteMessage(websocket.TextMessage, []byte(c.Marshal()))
			if err != nil {
				log.Println("write:", err)
			}
		}
	})

	// 监听信令服务器的消息
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			candidate, err := ice.UnmarshalCandidate(string(message))
			if err != nil {
				log.Println("unmarshal:", err)
				continue
			}
			err = agent.AddRemoteCandidate(candidate)
			if err != nil {
				log.Println("AddRemoteCandidate:", err)
			}
		}
	}()

	// 使用硬编码的用户名和密码创建 ICE 连接
	iceConn, err := agent.Dial(context.Background(), hardcodedUfrag, hardcodedPwd)
	if err != nil {
		log.Fatal("Failed to dial ICE connection:", err)
	}

	// 发送消息
	go sendMessage(iceConn)

	// 接收消息
	go receiveMessage(iceConn)

	// 等待 Ctrl+C 退出
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	fmt.Println("Exiting...")
	iceConn.Close()
	agent.Close()
	conn.Close()
}

func sendMessage(iceConn *ice.Conn) {
	for {
		message := "Hello from client!"
		_, err := iceConn.Write([]byte(message))
		if err != nil {
			log.Println("send error:", err)
			return
		}
		fmt.Printf("Sent message: %s\n", message)
	}
}

func receiveMessage(iceConn *ice.Conn) {
	buf := make([]byte, 1500) // 数据报最大长度
	for {
		n, err := iceConn.Read(buf)
		if err != nil {
			log.Println("read error:", err)
			return
		}
		fmt.Printf("Received message: %s\n", string(buf[:n]))
	}
}
