package publisher

import (
	"context"
	"errors"
	"github.com/cossteam/punchline/api/v1"
	"go.uber.org/zap"
	"sync"
	"time"
)

var _ api.PubSubServiceServer = &PubsubService{}

func NewPubsubService(logger *zap.Logger) *PubsubService {
	return &PubsubService{
		logger:           logger,
		pub:              NewPublisher(100*time.Millisecond, 10),
		topicSubscribers: make(map[string]map[string]chan interface{}),
		unsubCh:          make(chan interface{}, 100),
	}
}

type PubsubService struct {
	pub              Publisher
	logger           *zap.Logger
	topicSubscribers map[string]map[string]chan interface{} // Map of topic to map of hostname to channel
	mu               sync.RWMutex
	unsubCh          chan interface{}
}

func (ps *PubsubService) Unsubscribe(ctx context.Context, request *api.UnsubscribeRequest) (*api.UnsubscribeResponse, error) {
	if request.Topic == "" || request.Hostname == "" {
		return nil, errors.New("invalid unsubscribe request: missing Topic or Hostname")
	}

	ps.cleaning(request.Topic, request.Hostname, func(ch chan interface{}) {
		ps.unsubCh <- ch
	})

	//ps.mu.Lock()
	//defer ps.mu.Unlock()
	//
	//if subscribers, ok := ps.topicSubscribers[request.Topic]; ok {
	//	if ch, ok := subscribers[request.Hostname]; ok {
	//		delete(subscribers, request.Hostname)
	//		ps.unsubCh <- ch
	//		if len(subscribers) == 0 {
	//			delete(ps.topicSubscribers, request.Topic)
	//		}
	//		ps.logger.Debug("取消订阅成功", zap.String("topic", request.Topic), zap.String("hostname", request.Hostname))
	//	} else {
	//		ps.logger.Warn("未找到订阅者", zap.String("topic", request.Topic), zap.String("hostname", request.Hostname))
	//	}
	//} else {
	//	ps.logger.Warn("未找到主题订阅", zap.String("topic", request.Topic))
	//}

	return &api.UnsubscribeResponse{}, nil
}

func (ps *PubsubService) Publish(ctx context.Context, req *api.PublishRequest) (*api.PublishResponse, error) {
	ps.logger.Debug("收到发布请求", zap.String("topic", req.Topic))
	ps.pub.Publish(&api.Message{
		Topic: req.Topic,
		//Event: req.Event,
		Data: req.Data,
	})
	return &api.PublishResponse{}, nil
}

func (ps *PubsubService) Subscribe(req *api.SubscribeRequest, stream api.PubSubService_SubscribeServer) error {
	if req.Topic == "" {
		return errors.New("invalid subscribe request: missing Topic")
	}

	ch := ps.pub.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(*api.Message); ok {
			return key.Topic == req.Topic
		}
		return false
	})

	ps.logger.Debug("收到订阅请求", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))

	ps.mu.Lock()
	if ps.topicSubscribers[req.Topic] == nil {
		ps.topicSubscribers[req.Topic] = make(map[string]chan interface{})
	}
	ps.topicSubscribers[req.Topic][req.Hostname] = ch
	ps.mu.Unlock()

	for {
		select {
		case <-stream.Context().Done():
			ps.logger.Debug("取消订阅1", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))
			ps.pub.Evict(ch)
			ps.cleaning(req.Topic, req.Hostname, nil)
			return nil
		case v := <-ps.unsubCh:
			if v == ch {
				ps.logger.Debug("取消订阅2", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))
				ps.pub.Evict(ch)
				return nil
			}
		case v := <-ch:
			if msg, ok := v.(*api.Message); ok {
				if err := stream.Send(&api.Message{
					Topic: msg.Topic,
					//Event: msg.Event,
					Data: msg.Data,
				}); err != nil {
					ps.logger.Error("发送消息失败", zap.Error(err))
					return err
				}
			}
		}
	}
}

func (ps *PubsubService) cleaning(topic, hostname string, f func(ch chan interface{})) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if subscribers, ok := ps.topicSubscribers[topic]; ok {
		if ch, ok := subscribers[hostname]; ok {
			delete(subscribers, hostname)
			if f != nil {
				f(ch)
			}
			if len(subscribers) == 0 {
				delete(ps.topicSubscribers, topic)
			}
			ps.logger.Debug("取消订阅成功", zap.String("topic", topic), zap.String("hostname", hostname))
		} else {
			ps.logger.Warn("未找到订阅者", zap.String("topic", topic), zap.String("hostname", hostname))
		}
	} else {
		ps.logger.Warn("未找到主题订阅", zap.String("topic", topic))
	}
}
