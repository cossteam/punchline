package publisher

import (
	"context"
	"github.com/cossteam/punchline/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"net"
	"sync/atomic"
)

var _ PublisherClient = &client{}

type ClientOption interface {
	apply(*client)
}

// WithClientName 返回一个设置客户端名称的选项
func WithClientName(name string) ClientOption {
	return clientOptionFunc(func(c *client) {
		c.hostname = name
	})
}

// clientOptionFunc 是一个实现 ClientOption 接口的函数类型
type clientOptionFunc func(*client)

func (f clientOptionFunc) apply(c *client) {
	f(c)
}

func NewClient(addr string, opts ...ClientOption) (PublisherClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c := &client{
		pss:  api.NewPubSubServiceClient(conn),
		conn: conn,
		//subscribeStreams: make(map[string]grpc.ClientStream),
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c, err
}

// NewClientWithDialer 使用 bufDialer 来创建一个 gRPC 客户端
func NewClientWithDialer(bufDialer func(context.Context, string) (net.Conn, error), opts ...ClientOption) (PublisherClient, error) {
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	c := &client{
		pss:  api.NewPubSubServiceClient(conn),
		conn: conn,
		//subscribeStreams: make(map[string]grpc.ClientStream),
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c, nil
}

type client struct {
	hostname string
	conn     *grpc.ClientConn
	pss      api.PubSubServiceClient

	closed atomic.Bool
	//subscribeStreams map[string]grpc.ClientStream
	//unsubCh          chan interface{}
	//mu               sync.Mutex
}

func (c *client) Close() error {
	c.closed.Store(true)
	return c.conn.Close()
}

func (c *client) Publish(ctx context.Context, message *Message) error {
	_, err := c.pss.Publish(context.Background(), &api.PublishRequest{
		Topic:    message.Topic,
		Hostname: c.hostname,
		Data:     message.Data,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *client) Subscribe(ctx context.Context, topic string, handler func(*Message) error) error {
	stream, err := c.pss.Subscribe(context.Background(), &api.SubscribeRequest{
		Topic:    topic,
		Hostname: c.hostname,
	})
	if err != nil {
		return err
	}

	//c.mu.Lock()
	//if c.subscribeStreams == nil {
	//	c.subscribeStreams = make(map[string]grpc.ClientStream)
	//}
	//c.subscribeStreams[topic] = stream
	//c.mu.Unlock()

	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				if err == io.EOF || c.closed.Load() {
					break
				}
				log.Printf("error receiving message from stream: %v", err)
				break
			}

			msg := &Message{
				Topic: res.Topic,
				//Event: Event(res.Event),
				Data: res.Data,
			}
			if err := handler(msg); err != nil {
				log.Printf("error handling message: %v", err)
			}
		}
	}()

	return nil
}

func (c *client) Unsubscribe(ctx context.Context, topic string) error {
	_, err := c.pss.Unsubscribe(context.Background(), &api.UnsubscribeRequest{
		Topic:    topic,
		Hostname: c.hostname,
	})
	return err
}
