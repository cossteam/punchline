package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/ice/v2"
)

func main() {
	// 使用 flag 来指定信令服务器地址
	serverAddr := flag.String("addr", "localhost:8080", "signaling server address")
	flag.Parse()

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+*serverAddr+"/ws", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	agent, err := ice.NewAgent(&ice.AgentConfig{})
	if err != nil {
		log.Fatal(err)
	}

	// 处理候选者收集事件
	agent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			fmt.Println("Discovered new candidate:", c.String())
			err := conn.WriteMessage(websocket.TextMessage, []byte(c.Marshal()))
			if err != nil {
				log.Println("write:", err)
			}
		}
	})

	// 处理ICE连接状态改变事件
	agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Println("Connection State has changed:", state.String())
		if state == ice.ConnectionStateConnected {
			fmt.Println("Connected to remote peer")
			go sendMessage(agent)
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

func sendMessage(agent *ice.Agent) {
	for {
		time.Sleep(5 * time.Second)
		err := agent.Send([]byte("Hello from client!"))
		if err != nil {
			log.Println("send error:", err)
			return
		}
	}
}
