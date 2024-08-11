package signaling

import (
	"context"
	"errors"
	apiv1 "github.com/cossteam/punchline/api"
	"github.com/cossteam/punchline/api/signaling/v1"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/publisher"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"sync"
	"time"
)

var _ signaling.SignalingServer = &SignalingController{}
var _ apiv1.Runnable = &SignalingController{}

type SignalingController struct {
	addr   string
	server *grpc.Server

	pub              publisher.Publisher
	logger           *zap.Logger
	topicSubscribers map[string]map[string]chan interface{} // Map of topic to map of hostname to channel
	mu               sync.RWMutex
	unsubCh          chan interface{}
}

func NewSignalingController(addr string, logger *zap.Logger, opts ...grpc.ServerOption) *SignalingController {
	sc := &SignalingController{
		addr:             addr,
		server:           grpc.NewServer(opts...),
		logger:           logger.With(zap.String("controller", "signaling")),
		pub:              publisher.NewPublisher(100*time.Millisecond, 10),
		topicSubscribers: make(map[string]map[string]chan interface{}),
		unsubCh:          make(chan interface{}, 100),
	}

	signaling.RegisterSignalingServer(sc.server, sc)

	return sc
}

func (sc *SignalingController) Start(ctx context.Context) error {
	serverShutdown := make(chan struct{})
	go func() {
		<-ctx.Done()
		sc.logger.Info("Shutting down SignalingServer")
		sc.server.GracefulStop()
		close(serverShutdown)
	}()

	gaddr := sc.addr
	lis, err := net.Listen("tcp", gaddr)
	if err != nil {
		return err
	}

	sc.logger.Info("Starting SignalingServer", zap.Any("addr", gaddr))

	go func() {
		if err := sc.server.Serve(lis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				sc.logger.Error("Failed to serve SignalingServer", zap.Error(err))
			}
		}
	}()

	<-serverShutdown

	return nil
}

func (sc *SignalingController) Unsubscribe(ctx context.Context, request *api.UnsubscribeRequest) (*api.UnsubscribeResponse, error) {
	if request.Topic == "" || request.Hostname == "" {
		return nil, errors.New("invalid unsubscribe request: missing Topic or Hostname")
	}

	sc.cleaning(request.Topic, request.Hostname, func(ch chan interface{}) {
		sc.unsubCh <- ch
	})

	return &api.UnsubscribeResponse{}, nil
}

func (sc *SignalingController) Publish(ctx context.Context, req *signaling.PublishRequest) (*signaling.PublishResponse, error) {
	sc.logger.Debug("收到发布请求", zap.String("topic", req.Topic), zap.Any("candidate", req.Candidate))
	sc.pub.Publish(&signaling.Message{
		Topic:       req.Topic,
		Data:        req.Data,
		Credentials: req.Credentials,
		Candidate:   req.Candidate,
	})
	return &signaling.PublishResponse{}, nil
}

func (sc *SignalingController) Subscribe(req *signaling.SubscribeRequest, stream signaling.Signaling_SubscribeServer) error {
	if req.Topic == "" {
		return errors.New("invalid subscribe request: missing Topic")
	}

	ch := sc.pub.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(*signaling.Message); ok {
			return key.Topic == req.Topic
		}
		return false
	})

	sc.logger.Debug("收到订阅请求", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))

	sc.mu.Lock()
	if sc.topicSubscribers[req.Topic] == nil {
		sc.topicSubscribers[req.Topic] = make(map[string]chan interface{})
	}
	sc.topicSubscribers[req.Topic][req.Hostname] = ch
	sc.mu.Unlock()

	for {
		select {
		case <-stream.Context().Done():
			sc.logger.Debug("取消订阅1", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))
			sc.pub.Evict(ch)
			sc.cleaning(req.Topic, req.Hostname, nil)
			return nil
		case v := <-sc.unsubCh:
			if v == ch {
				sc.logger.Debug("取消订阅2", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))
				sc.pub.Evict(ch)
				return nil
			}
		case v := <-ch:
			if msg, ok := v.(*signaling.Message); ok {
				sc.logger.Debug("发送消息", zap.String("topic", msg.Topic), zap.Any("message", msg))
				if err := stream.Send(&signaling.Message{
					Topic: msg.Topic,
					Data:  msg.Data,

					Credentials: msg.Credentials,
					Candidate:   msg.Candidate,
				}); err != nil {
					sc.logger.Error("发送消息失败", zap.Error(err))
					return err
				}
			}
		}
	}
}

func (sc *SignalingController) cleaning(topic, hostname string, f func(ch chan interface{})) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if subscribers, ok := sc.topicSubscribers[topic]; ok {
		if ch, ok := subscribers[hostname]; ok {
			delete(subscribers, hostname)
			if f != nil {
				f(ch)
			}
			if len(subscribers) == 0 {
				delete(sc.topicSubscribers, topic)
			}
			sc.logger.Debug("取消订阅成功", zap.String("topic", topic), zap.String("hostname", hostname))
		} else {
			sc.logger.Warn("未找到订阅者", zap.String("topic", topic), zap.String("hostname", hostname))
		}
	} else {
		sc.logger.Warn("未找到主题订阅", zap.String("topic", topic))
	}
}
