package signal

import (
	"context"
	"github.com/cossteam/punchline/api/signaling/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"net"
	"sync/atomic"
)

type (
	Message     = signaling.Message
	Candidate   = signaling.Candidate
	Credentials = signaling.Credentials
)

var _ Client = &SignalingClient{}

type SignalingClient struct {
	hostname string
	conn     *grpc.ClientConn
	signal   signaling.SignalingClient

	closed atomic.Bool
	//subscribeStreams map[string]grpc.ClientStream
	//unsubCh          chan interface{}
	//mu               sync.Mutex
}

func NewClient(addr string, opts ...ClientOption) (*SignalingClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	c := &SignalingClient{
		signal: signaling.NewSignalingClient(conn),
		conn:   conn,
		//subscribeStreams: make(map[string]grpc.ClientStream),
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c, err
}

// NewClientWithDialer 本地调试使用 bufDialer 来创建一个 gRPC 客户端
func NewClientWithDialer(bufDialer func(context.Context, string) (net.Conn, error), opts ...ClientOption) (*SignalingClient, error) {
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	c := &SignalingClient{
		signal: signaling.NewSignalingClient(conn),
		conn:   conn,
		//subscribeStreams: make(map[string]grpc.ClientStream),
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c, nil
}

func (c *SignalingClient) Close() error {
	c.closed.Store(true)
	return c.conn.Close()
}

func (c *SignalingClient) Publish(ctx context.Context, message *Message) error {
	_, err := c.signal.Publish(ctx, &signaling.PublishRequest{
		Topic:    message.Topic,
		Hostname: c.hostname,
		Data:     message.Data,

		Credentials: message.Credentials,
		Candidate:   message.Candidate,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *SignalingClient) Subscribe(ctx context.Context, topic string, handler func(*Message) error) error {
	stream, err := c.signal.Subscribe(ctx, &signaling.SubscribeRequest{
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

func (c *SignalingClient) Unsubscribe(ctx context.Context, topic string) error {
	//_, err := c.signal.Unsubscribe(context.Background(), &api.UnsubscribeRequest{
	//	Topic:    topic,
	//	Hostname: c.hostname,
	//})
	return nil
}
