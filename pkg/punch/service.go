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

func (ps *punchService) HostOnline(ctx context.Context, request *api.HostOnlineRequest) (*api.HostOnlineResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ps *punchService) HostQuery(ctx context.Context, request *api.HostQueryRequest) (*api.HostQueryResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ps *punchService) HostUpdate(ctx context.Context, request *api.HostUpdateRequest) (*api.HostUpdateResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ps *punchService) HostPunch(ctx context.Context, request *api.HostPunchRequest) (*api.HostPunchResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ps *punchService) HostMoved(ctx context.Context, request *api.HostMovedRequest) (*api.HostMovedResponse, error) {
	//TODO implement me
	panic("implement me")
}
