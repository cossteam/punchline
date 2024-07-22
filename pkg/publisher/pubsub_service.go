package publisher

import (
	"context"
	"errors"
	"github.com/cossteam/punchline/api/v1"
	"go.uber.org/zap"
	"time"
)

var _ api.PubSubServiceServer = &PubsubService{}

type PubsubService struct {
	pub    *publisher
	logger *zap.Logger
}

func NewPubsubService(logger *zap.Logger) *PubsubService {
	return &PubsubService{
		logger: logger,
		pub:    NewPublisher(100*time.Millisecond, 10),
	}
}

func (p *PubsubService) Publish(ctx context.Context, req *api.PublishRequest) (*api.PublishResponse, error) {
	p.pub.Publish(&api.Message{
		Topic: req.Topic,
		//Event: req.Event,
		Data: req.Data,
	})
	return &api.PublishResponse{}, nil
}

func (p *PubsubService) Subscribe(req *api.SubscribeRequest, stream api.PubSubService_SubscribeServer) error {
	if req.Topic == "" {
		return errors.New("invalid subscribe request: missing Topic")
	}

	ch := p.pub.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(*api.Message); ok {
			return key.Topic == req.Topic
		}
		return false
	})

	p.logger.Debug("收到订阅请求", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))

	for {
		select {
		case <-stream.Context().Done():
			p.logger.Debug("取消订阅", zap.String("topic", req.Topic), zap.String("hostname", req.Hostname))
			return nil
		case v := <-ch:
			if msg, ok := v.(*api.Message); ok {
				if err := stream.Send(&api.Message{
					Topic: msg.Topic,
					//Event: msg.Event,
					Data: msg.Data,
				}); err != nil {
					p.logger.Error("发送消息失败", zap.Error(err))
					return err
				}
			}
		}
	}
}
