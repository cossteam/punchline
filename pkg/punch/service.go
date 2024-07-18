package punch

import (
	"context"
	api "github.com/cossteam/punchline/api/v1"
)

var _ api.PunchServiceServer = &punchService{}

type punchService struct {
}

func (p *punchService) HostQuery(ctx context.Context, request *api.HostQueryRequest) (*api.HostQueryResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *punchService) HostUpdate(ctx context.Context, request *api.HostUpdateRequest) (*api.HostUpdateResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *punchService) HostPunch(ctx context.Context, request *api.HostPunchRequest) (*api.HostPunchResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *punchService) HostMoved(ctx context.Context, request *api.HostMovedRequest) (*api.HostMovedResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *punchService) HostSubscribe(ctx context.Context, request *api.HostSubscribeRequest) (*api.HostSubscribeResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (p *punchService) mustEmbedUnimplementedPunchServiceServer() {
	//TODO implement me
	panic("implement me")
}
