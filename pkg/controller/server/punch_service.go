package controller

import (
	"context"
	"errors"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/host"
	"go.uber.org/zap"
)

var _ api.PunchServiceServer = &serverController{}

func (sc *serverController) HostOnline(ctx context.Context, request *api.HostOnlineRequest) (*api.HostOnlineResponse, error) {
	sc.logger.Debug("主机上线通知", zap.Any("request", request))

	hostname := request.Hostname

	newHm := &api.HostMessage{}
	found, ln, err := sc.queryAndPrepMessage(hostname, func(cache *host.Cache) (int, error) {
		newHm.Type = api.HostMessage_HostOnlineNotification
		newHm.Hostname = hostname
		newHm.ExternalAddr = request.ExternalAddr
		sc.coalesceAnswers(cache, newHm)
		return newHm.MarshalTo(sc.p)
	})
	if !found {
		sc.logger.Debug("未找到主机信息", zap.String("name", hostname))
		return nil, err
	}
	if err != nil {
		sc.logger.Error("Failed to marshal lighthouse host query reply", zap.String("name", hostname))
		return nil, err
	}

	if _, err = sc.Publish(context.Background(), &api.PublishRequest{
		Topic: hostname,
		Data:  sc.p[:ln],
	}); err != nil {
		sc.logger.Error("Failed to publish lighthouse host update ack",
			zap.String("hostname", hostname),
			zap.Error(err),
		)
		return nil, err
	}

	return &api.HostOnlineResponse{}, nil
}

func (sc *serverController) HostQuery(ctx context.Context, request *api.HostQueryRequest) (*api.HostQueryResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (sc *serverController) HostUpdate(ctx context.Context, request *api.HostUpdateRequest) (*api.HostUpdateResponse, error) {
	hostname := request.Hostname

	var hostInfo *host.HostInfo
	hostInfo = sc.GetOrCreateHostInfo(hostname)

	//hostInfo.SetRemote(addr)
	//fmt.Println("hostInfo.Remote => ", hostInfo.Remote)
	//fmt.Println("addr => ", addr)

	oldAddr := hostInfo.Remotes.CopyAddrs()
	//hostInfo.SetRemote(addr)

	sc.Lock()
	am := sc.unlockedGetRemoteList(hostname)
	am.Lock()
	sc.Unlock()
	am.UnlockedSetV4(hostname, request.Ipv4Addr)
	am.UnlockedSetV6(hostname, request.Ipv6Addr)
	am.Unlock()
	newAddr := am.CopyAddrs()

	newHm := &api.HostMessage{}
	found, ln, err := sc.queryAndPrepMessage(hostname, func(cache *host.Cache) (int, error) {
		newHm.Type = api.HostMessage_HostPunchNotification
		newHm.Hostname = hostname
		newHm.ExternalAddr = request.ExternalAddr
		sc.coalesceAnswers(cache, newHm)
		return newHm.MarshalTo(sc.p)
	})
	if !found {
		sc.logger.Debug("未找到主机信息", zap.String("hostname", hostname))
		return nil, errors.New("未找到主机信息")
	}

	if err != nil {
		sc.logger.Error("Failed to marshal lighthouse host query reply", zap.String("hostname", hostname))
		return nil, err
	}

	sc.logger.Debug("收到主机更新通知",
		zap.String("handle", "HostUpdate"),
		//zap.Any("oldHm", request),
		//zap.Any("newHm", newHm),
		zap.Any("oldAddr", oldAddr),
		zap.Any("newAddr", newAddr),
	)

	//hm.Reset()
	//hm.Type = api.HostMessage_None
	//hm.Type = api.HostMessage_HostPunchNotification
	//hm.Hostname = sc.hostname
	//ln, err := hm.MarshalTo(sc.p)
	//if err != nil {
	//	sc.logger.Error("Failed to marshal lighthouse host update ack",
	//		zap.String("hostname", hm.Hostname),
	//	)
	//	return
	//}

	if hasAddressChanged(oldAddr, newAddr) {
		sc.logger.Info("地址发送变化，开始推送",
			zap.String("handle", "HostUpdate"),
			zap.Any("topic", hostname),
		)

		_, err = sc.Publish(context.Background(), &api.PublishRequest{
			Topic: hostname,
			Data:  sc.p[:ln],
		})
		if err != nil {
			sc.logger.Error("Failed to publish lighthouse host update ack",
				zap.String("hostname", hostname),
				zap.Error(err),
			)
		}
	}

	return &api.HostUpdateResponse{}, nil
}

func (sc *serverController) HostPunch(ctx context.Context, request *api.HostPunchRequest) (*api.HostPunchResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (sc *serverController) HostMoved(ctx context.Context, request *api.HostMovedRequest) (*api.HostMovedResponse, error) {
	//TODO implement me
	panic("implement me")
}
