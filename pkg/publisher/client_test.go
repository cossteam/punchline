package publisher

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/cossteam/punchline/api/v1"
	"github.com/stretchr/testify/assert"
)

type MockHostSubscribeClient struct{}

func (m *MockHostSubscribeClient) Recv() (*api.HostSubscribeResponse, error) {
	// mock 实现
	return nil, io.EOF
}

func (m *MockHostSubscribeClient) CloseSend() error {
	// mock 实现
	return nil
}

var (
	Hostname = "test-hostname"
	Ipv4Addr = []*api.Ipv4Addr{
		api.NewIpv4Addr(net.ParseIP("192.168.1.1"), 8080),
		api.NewIpv4Addr(net.ParseIP("192.168.1.2"), 8081),
		api.NewIpv4Addr(net.ParseIP("192.168.1.3"), 8082),
	}

	Ipv6Addr = []*api.Ipv6Addr{
		api.NewIpv6Addr(net.ParseIP("2001:db8:85a3:08d3:1319:8a2e:0370:7344"), 8080),
		api.NewIpv6Addr(net.ParseIP("2001:db8:85a3:08d3:1319:8a2e:0370:7345"), 8081),
	}
)

func TestClient_Subscribe(t *testing.T) {
	c, err := NewClient("localhost:7777", WithClientName(Hostname))
	assert.NoError(t, err)

	handlerCalled := false
	handler := func(msg *Message) error {
		handlerCalled = true
		fmt.Println("subscribe handler msg Data => ", msg.Data)
		return nil
	}

	err = c.Subscribe(context.Background(), "test-topic", handler)
	assert.NoError(t, err)

	// 等待 goroutine 处理消息
	time.Sleep(3 * time.Second)
	assert.True(t, handlerCalled)
}

func TestClient_Publish(t *testing.T) {
	c, err := NewClient("localhost:7777", WithClientName(Hostname))
	assert.NoError(t, err)

	m := &api.HostMessage{
		Type: api.HostMessage_HostUpdateNotification,
		//Ipv4Addr: Ipv4Addr,
		//Ipv6Addr: Ipv6Addr,
		Hostname: Hostname,
	}

	marshal, err := m.Marshal()
	assert.NoError(t, err)

	err = c.Publish(context.Background(), &Message{
		Topic: "client-2",
		Data:  marshal,
	})
	assert.NoError(t, err)
}

//func TestClient_Close(t *testing.T) {
//	conn, err := grpc.Dial("localhost:7777", grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		log.Fatalf("did not connect: %v", err)
//	}
//	defer conn.Close()
//
//	mockPunchServiceClient := api.NewPunchServiceClient(conn)
//	c := NewClient(context.Background(), "test-hostname", mockPunchServiceClient)
//
//	err = c.Close()
//	assert.Error(t, errors.New("implement me"), err)
//}
//
//func TestClient_Unsubscribe(t *testing.T) {
//	conn, err := grpc.Dial("localhost:7777", grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		log.Fatalf("did not connect: %v", err)
//	}
//	defer conn.Close()
//
//	mockPunchServiceClient := api.NewPunchServiceClient(conn)
//	c := NewClient(context.Background(), "test-hostname", mockPunchServiceClient)
//
//	err = c.Unsubscribe("test-topic")
//	assert.NoError(t, err)
//}
