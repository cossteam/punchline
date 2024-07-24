package controller

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	apiv1 "github.com/cossteam/punchline/api"
	"github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/config"
	"github.com/cossteam/punchline/pkg/host"
	"github.com/cossteam/punchline/pkg/publisher"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"sync"
)

var (
	_ apiv1.Runnable = &serverController{}

	_ api.PubSubServiceServer = &serverController{}

	//_ pubsub.PubSubService = &serverController{}
)

func NewServerController(
	logger *zap.Logger,
	listenPort uint32,
	hostname string,
	outside udp.Conn,
	c *config.Config,
) apiv1.Runnable {
	return &serverController{
		server:     grpc.NewServer(),
		logger:     logger,
		listenPort: listenPort,
		hostname:   hostname,
		outside:    outside,
		c:          c,

		pubSvc: publisher.NewPubsubService(logger),

		hostMap: host.NewHostMap(logger),
		addrMap: make(map[string]*host.RemoteList),
		p:       make([]byte, mtu),
		out:     make([]byte, mtu),
	}
}

type serverController struct {
	sync.RWMutex

	c          *config.Config
	logger     *zap.Logger
	listenPort uint32
	hostname   string
	outside    udp.Conn

	server    *grpc.Server
	publisher publisher.Publisher
	pubSvc    api.PubSubServiceServer

	addrMap map[string]*host.RemoteList
	hostMap *host.HostMap

	p   []byte
	out []byte
}

func (sc *serverController) Unsubscribe(ctx context.Context, request *api.UnsubscribeRequest) (*api.UnsubscribeResponse, error) {
	return &api.UnsubscribeResponse{}, nil
}

func (sc *serverController) Publish(ctx context.Context, request *api.PublishRequest) (*api.PublishResponse, error) {
	//fmt.Println("serverController Publish => ", request)
	//err := sc.pubSvc.Publish(ctx, &publisher.Message{
	//	Topic: request.Topic,
	//	Data:  request.Data,
	//})
	//if err != nil {
	//	return nil, err
	//}
	return sc.pubSvc.Publish(ctx, request)
}

func (sc *serverController) Subscribe(request *api.SubscribeRequest, subscribeServer api.PubSubService_SubscribeServer) error {
	return nil
}

func (sc *serverController) Start(ctx context.Context) error {
	serverShutdown := make(chan struct{})
	go func() {
		<-ctx.Done()
		sc.logger.Info("Shutting down Server")
		if err := sc.outside.Close(); err != nil {
			sc.logger.Error("Failed to close Server", zap.Error(err))
		}
		sc.server.Stop()
		close(serverShutdown)
	}()

	addr, err := sc.outside.LocalAddr()
	if err != nil {
		return err
	}

	go func() {
		sc.logger.Info("Starting Server", zap.Any("addr", addr))
		sc.listenOutside()
	}()

	gaddr := fmt.Sprintf("0.0.0.0:%d", sc.listenPort)
	lis, err := net.Listen("tcp", gaddr)
	if err != nil {
		return err
	}

	api.RegisterPubSubServiceServer(sc.server, sc.pubSvc)

	go func() {
		sc.logger.Info("Starting grpcServer", zap.Any("addr", gaddr))
		if err := sc.server.Serve(lis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				sc.logger.Error("Failed to serve grpcServer", zap.Error(err))
			}
		}
	}()

	<-serverShutdown

	return nil
}

func (sc *serverController) listenOutside() {
	sc.outside.Listen(func(addr *udp.Addr, out []byte, packet []byte) {
		sc.HandleRequest(addr.Copy(), packet)
	})
}

func (sc *serverController) HandleRequest(addr *udp.Addr, p []byte) {
	//fmt.Println("p => ", p)
	//fmt.Println("p len => ", len(p))
	hm := &api.HostMessage{}
	if err := hm.Unmarshal(p); err != nil {
		sc.logger.Error("Failed to unmarshal lighthouse packet",
			zap.Error(err),
		)
		//TODO: send recv_error?
		return
	}

	var hostInfo *host.HostInfo
	hostInfo = sc.GetOrCreateHostInfo(hm.Hostname)

	//printUDPHeader(p)

	switch hm.Type {
	case api.HostMessage_HostQuery:

	case api.HostMessage_HostQueryReply:

	case api.HostMessage_HostUpdateNotification:
		sc.handleHostUpdateNotification(hm, addr, hostInfo)
	case api.HostMessage_HostMovedNotification:

	case api.HostMessage_HostOnlineNotification:
		sc.handleHostOnlineNotification(hm, addr)
	}
}

func (sc *serverController) handleHostUpdateNotification(hm *api.HostMessage, addr *udp.Addr, hostInfo *host.HostInfo) {
	name := hm.Hostname

	//hostInfo.SetRemote(addr)
	fmt.Println("hostInfo.Remote => ", hostInfo.Remote)
	fmt.Println("addr => ", addr)

	oldAddr := hostInfo.Remotes.CopyAddrs()
	//hostInfo.SetRemote(addr)

	sc.Lock()
	am := sc.unlockedGetRemoteList(name)
	am.Lock()
	sc.Unlock()
	am.UnlockedSetV4(name, hm.Ipv4Addr)
	am.UnlockedSetV6(name, hm.Ipv6Addr)
	am.Unlock()
	newAddr := am.CopyAddrs()

	newHm := &api.HostMessage{}
	found, ln, err := sc.queryAndPrepMessage(name, func(cache *host.Cache) (int, error) {
		newHm.Type = api.HostMessage_HostPunchNotification
		newHm.Hostname = name
		newHm.ExternalAddr = hm.ExternalAddr
		sc.coalesceAnswers(cache, newHm)
		return newHm.MarshalTo(sc.p)
	})
	if !found {
		sc.logger.Debug("未找到主机信息", zap.String("name", name))
		return
	}

	if err != nil {
		sc.logger.Error("Failed to marshal lighthouse host query reply", zap.String("name", name))
		return
	}

	sc.logger.Info("收到主机更新通知",
		zap.Any("oldHm", hm),
		zap.Any("newHm", newHm),
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
			zap.Any("topic", name),
			zap.Any("oldAddr", oldAddr),
			zap.Any("newAddr", newAddr))

		_, err = sc.Publish(context.Background(), &api.PublishRequest{
			Topic: name,
			Data:  sc.p[:ln],
		})
		if err != nil {
			sc.logger.Error("Failed to publish lighthouse host update ack",
				zap.String("hostname", hm.Hostname),
				zap.Error(err),
			)
		}
	}

	//if err := sc.outside.WriteTo(sc.p[:ln], addr); err != nil {
	//	sc.logger.Error("Failed to send lighthouse host update ack",
	//		zap.String("hostname", hm.Hostname),
	//		zap.Error(err),
	//	)
	//}
}

func hasAddressChanged(oldAddrs, newAddrs []*udp.Addr) bool {
	if len(oldAddrs) != len(newAddrs) {
		return true
	}

	oldAddrMap := make(map[string]struct{}, len(oldAddrs))
	for _, addr := range oldAddrs {
		oldAddrMap[addr.String()] = struct{}{}
	}

	for _, addr := range newAddrs {
		if _, exists := oldAddrMap[addr.String()]; !exists {
			return true
		}
	}

	return false
}

func (sc *serverController) unlockedGetRemoteList(name string) *host.RemoteList {
	am, ok := sc.addrMap[name]
	if !ok {
		am = host.NewRemoteList()
		sc.addrMap[name] = am
	}
	return am
}

func (sc *serverController) handleHostOnlineNotification(hm *api.HostMessage, addr *udp.Addr) {
	sc.logger.Info("收到主机上线通知", zap.Any("hm", hm), zap.Any("addr", addr))
}

// GetOrCreateHostInfo retrieves the existing HostInfo or creates a new one if it doesn't exist.
func (sc *serverController) GetOrCreateHostInfo(hostname string) *host.HostInfo {
	hostInfo := sc.hostMap.GetHost(hostname)
	if hostInfo == nil {
		hostInfo = &host.HostInfo{
			Name:    hostname,
			Remotes: sc.unlockedGetRemoteList(hostname),
		}
		sc.hostMap.AddHost(hostInfo)
	}
	return hostInfo
}

func printUDPHeader(header []byte) {
	if len(header) < 8 {
		fmt.Println("Invalid UDP header")
		return
	}
	srcPort := binary.BigEndian.Uint16(header[0:2])
	destPort := binary.BigEndian.Uint16(header[2:4])
	length := binary.BigEndian.Uint16(header[4:6])
	checksum := binary.BigEndian.Uint16(header[6:8])

	fmt.Printf("UDP Header:\n")
	fmt.Printf("Source Port: %d\n", srcPort)
	fmt.Printf("Destination Port: %d\n", destPort)
	fmt.Printf("Length: %d\n", length)
	fmt.Printf("Checksum: 0x%x\n", checksum)
}
