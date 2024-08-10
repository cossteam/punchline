package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/ice/v2"
	"log"
	"os"
	"os/signal"
)

var serverAddr = "ws://localhost:8080/ws"

func main() {
	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	agent, err := ice.NewAgent(&ice.AgentConfig{})
	if err != nil {
		log.Fatal(err)
	}

	// 处理候选者收集事件
	if err := agent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			fmt.Println("Discovered new candidate:", c.String())
			conn.WriteMessage(websocket.TextMessage, []byte(c.Marshal()))
		}
	}); err != nil {
		panic(err)
	}

	// 处理ICE连接状态改变事件
	if err := agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Println("Connection State has changed:", state.String())
		if state == ice.ConnectionStateConnected {
			fmt.Println("Connected to remote peer")
		}
	}); err != nil {
		panic(err)
	}

	// 监听信令服务器的消息
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			// 从字符串反序列化回候选者对象
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

	// 启动ICE连接
	err = agent.GatherCandidates()
	if err != nil {
		log.Fatal(err)
	}

	// 等待Ctrl+C退出
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	fmt.Println("Exiting...")
	agent.Close()
	conn.Close()
}
