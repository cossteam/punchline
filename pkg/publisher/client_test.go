package publisher

import (
	"context"
	"github.com/cossteam/punchline/api/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
	"time"
)

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

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func startGRPCServer() *grpc.Server {
	lis = bufconn.Listen(bufSize)
	server := grpc.NewServer()
	api.RegisterPubSubServiceServer(server, NewPubsubService(zap.NewNop()))

	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	return server
}

func TestPubsubService_Unsubscribe(t *testing.T) {
	server := startGRPCServer()
	defer server.Stop()

	c, err := NewClientWithDialer(bufDialer, WithClientName(Hostname))
	assert.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	err = c.Unsubscribe(ctx, "test-topic")
	assert.NoError(t, err)
}

func TestClient_Subscribe(t *testing.T) {
	server := startGRPCServer()
	defer server.Stop()

	c, err := NewClientWithDialer(bufDialer, WithClientName(Hostname))
	assert.NoError(t, err)
	defer c.Close()

	handler := func(msg *Message) error {
		return nil
	}

	err = c.Subscribe(context.Background(), "test-topic", handler)
	assert.NoError(t, err)
}

func TestClient_Publish(t *testing.T) {
	server := startGRPCServer()
	defer server.Stop()

	c, err := NewClientWithDialer(bufDialer, WithClientName(Hostname))
	assert.NoError(t, err)
	defer c.Close()

	m := &api.HostMessage{
		Type: api.HostMessage_HostUpdateNotification,
		//Ipv4Addr: Ipv4Addr,
		//Ipv6Addr: Ipv6Addr,
		Hostname: Hostname,
	}

	marshal, err := m.Marshal()
	assert.NoError(t, err)

	err = c.Publish(context.Background(), &Message{
		Topic: "test-topic",
		Data:  marshal,
	})
	assert.NoError(t, err)
}

func TestPubsubService_SubscribePublishUnsubscribe(t *testing.T) {
	server := startGRPCServer()
	defer server.Stop()

	c, err := NewClientWithDialer(bufDialer, WithClientName(Hostname))
	assert.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// 用于接收订阅消息的通道
	msgCh := make(chan *Message, 1)
	handler := func(msg *Message) error {
		msgCh <- msg
		return nil
	}

	// 订阅主题
	err = c.Subscribe(ctx, "test-topic", handler)
	assert.NoError(t, err)

	// 发布消息
	m := &api.HostMessage{
		Type:     api.HostMessage_HostUpdateNotification,
		Hostname: Hostname,
	}
	marshal, err := m.Marshal()
	assert.NoError(t, err)

	err = c.Publish(ctx, &Message{
		Topic: "test-topic",
		Data:  marshal,
	})
	assert.NoError(t, err)

	// 确保消息被接收
	select {
	case msg := <-msgCh:
		assert.Equal(t, "test-topic", msg.Topic)
		assert.Equal(t, marshal, msg.Data)
	case <-time.After(5 * time.Second):
		t.Fatal("消息接收超时")
	}

	// 取消订阅
	err = c.Unsubscribe(ctx, "test-topic")
	assert.NoError(t, err)
}
