package main

import (
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
)

func main() {
	serverAddr := flag.String("addr", "localhost:8080", "signaling server address")
	flag.Parse()

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+*serverAddr+"/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	agent, err := ice.NewAgent(&ice.AgentConfig{
		NetworkTypes: []ice.NetworkType{ice.NetworkTypeUDP4, ice.NetworkTypeUDP6},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := agent.SetRemoteCredentials(hardcodedUfrag, hardcodedPwd); err != nil {
		log.Fatal(err)
	}

	// 处理候选者收集事件
	if err := agent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			fmt.Println("Discovered new candidate:", c.String())
			err := conn.WriteMessage(websocket.TextMessage, []byte(c.Marshal()))
			if err != nil {
				log.Println("write:", err)
			}
		}
	}); err != nil {
		log.Fatal(err)
	}

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

	// 创建ICE连接
	connChannel := make(chan *ice.Conn)
	if err := agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Println("Connection State has changed:", state.String())
		if state == ice.ConnectionStateConnected {
			fmt.Println("Connected to remote peer")
			agent.OnSelectedCandidatePairChange(func(local, remote ice.Candidate) {
				iceConn, err := agent.Dial(nil, hardcodedUfrag, hardcodedPwd)
				if err != nil {
					log.Fatal("Failed to dial ICE connection:", err)
				}
				connChannel <- iceConn
			})
		}
	}); err != nil {
		log.Fatal(err)
	}

	// 启动 ICE 连接
	err = agent.GatherCandidates()
	if err != nil {
		log.Fatal(err)
	}

	// 获取 ICE 连接并发送消息
	iceConn := <-connChannel
	go sendMessage(iceConn)

	// 等待Ctrl+C退出
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
