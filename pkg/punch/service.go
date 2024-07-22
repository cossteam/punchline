package punch

import (
	"context"
	api "github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/publisher"
	"go.uber.org/zap"
)

var _ api.PunchServiceServer = &punchService{}

func NewPunchService(logger *zap.Logger, publisher publisher.Publisher) *punchService {
	return &punchService{
		logger:    logger,
		publisher: publisher,
	}
}

type punchService struct {
	logger    *zap.Logger
	publisher publisher.Publisher
}

func (p *punchService) HostSubscribe(request *api.HostSubscribeRequest, server api.PunchService_HostSubscribeServer) error {
	p.logger.Info("subscribe host", zap.String("hostname", request.Hostname))

	//go func() {
	//	p.publisher.Subscribe(request.Topic, request.Hostname, func(message *publisher.Message) error {
	//		toBytes, err := json.Marshal(message.Data)
	//		if err != nil {
	//			p.logger.Error("failed to marshal data", zap.Error(err))
	//			return err
	//		}
	//		return server.Send(&api.HostSubscribeResponse{
	//			Data:  string(toBytes),
	//			Event: message.Event.String(),
	//		})
	//	})
	//}()

	return nil
}

func (p *punchService) HostRegister(ctx context.Context, request *api.HostRegisterRequest) (*api.HostRegisterResponse, error) {
	p.logger.Info("register host", zap.String("hostname", request.Hostname))

	return &api.HostRegisterResponse{
		Success: true,
	}, nil
}

func (p *punchService) HostQuery(ctx context.Context, request *api.HostQueryRequest) (*api.HostQueryResponse, error) {
	//list := p.publisher.List()
	//for _, topic := range list {
	//	fmt.Println("topic => ", topic)
	//}
	return &api.HostQueryResponse{}, nil
}

func (p *punchService) HostUpdate(ctx context.Context, request *api.HostUpdateRequest) (*api.HostUpdateResponse, error) {
	p.logger.Info("update host", zap.String("hostname", request.Hostname))
	//marshal, err := request.Marshal()
	//if err != nil {
	//	return nil, err
	//}

	go func() {
		//if err := p.publisher.Publish(request.Hostname, publisher.Message{
		//	Event: "test",
		//	Data:  marshal,
		//}); err != nil {
		//	p.logger.Error("failed to publish", zap.Error(err))
		//}
	}()

	return &api.HostUpdateResponse{
		Success: true,
	}, nil
}

func (p *punchService) HostPunch(ctx context.Context, request *api.HostPunchRequest) (*api.HostPunchResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *punchService) HostMoved(ctx context.Context, request *api.HostMovedRequest) (*api.HostMovedResponse, error) {
	//TODO implement me
	panic("implement me")
}
