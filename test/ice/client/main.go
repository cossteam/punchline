package main

import (
	"context"
	"fmt"
	"github.com/pion/stun"
	"log"
	"os"
	"os/signal"

	"github.com/pion/ice/v2"
)

const (
	// 硬编码的 ICE 连接的用户名和密码
	hardcodedUfrag = "userfrag"
	hardcodedPwd   = "password"
	stunServer     = "stun.l.google.com:19302"
)

var (
	_iceURLs = []string{"stun:stun3.l.google.com:19302", "stun:stun.cunicu.li:3478", "stun:stun.easyvoip.com:3478"}
)

func main() {
	iceURLs, err := convertToStunURIs(_iceURLs)
	if err != nil {
		panic(err)
	}

	ac := &ice.AgentConfig{
		NetworkTypes: []ice.NetworkType{
			ice.NetworkTypeTCP4,
			ice.NetworkTypeTCP6,
			ice.NetworkTypeUDP4,
			ice.NetworkTypeUDP6,
		},
		Urls: iceURLs,
		CandidateTypes: []ice.CandidateType{
			ice.CandidateTypeHost,
			ice.CandidateTypeServerReflexive,
			ice.CandidateTypePeerReflexive,
			//ice.CandidateTypeRelay,
		},
	}

	// 创建 ICE agent，并使用 STUN 服务器配置
	agent, err := ice.NewAgent(ac)
	if err != nil {
		log.Fatal(err)
	}

	// 处理 ICE 候选者
	if err = agent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			fmt.Println("Discovered new candidate:", c.String())
		}
	}); err != nil {
		log.Fatal(err)
	}

	err = agent.GatherCandidates()
	if err != nil {
		panic(err)
	}

	// 使用硬编码的用户名和密码创建 ICE 连接
	iceConn, err := agent.Dial(context.Background(), hardcodedUfrag, hardcodedPwd)
	if err != nil {
		log.Fatal("Failed to dial ICE connection:", err)
	}

	fmt.Println("000")

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
	//conn.Close()
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

func convertToStunURIs(urls []string) ([]*stun.URI, error) {
	var iceURLs []*stun.URI
	for _, url := range urls {
		uri, err := stun.ParseURI(url)
		if err != nil {
			return nil, err
		}
		iceURLs = append(iceURLs, uri)
	}
	return iceURLs, nil
}
