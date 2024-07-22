package publisher

import (
	"context"
	"github.com/cossteam/punchline/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
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
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c, err
}

type client struct {
	hostname string
	conn     *grpc.ClientConn
	pss      api.PubSubServiceClient
}

func (c *client) Close() error {
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

	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
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
				// 处理处理函数中的错误
				log.Printf("error handling message: %v", err)
			}
		}
	}()

	return nil
}

func (c *client) Unsubscribe(ctx context.Context, topic string) error {
	//TODO implement me
	panic("implement me")
}
