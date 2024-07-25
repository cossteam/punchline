package plugin

import (
	"context"
	apiv1 "github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/config"
	"github.com/cossteam/punchline/pkg/utils"
	"go.uber.org/zap"
	"os/exec"
	"strings"
)

const (
	_name = "Wg"
)

// WGPlugin is a plugin that executes a shell command when a message is received.
type WGPlugin struct {
	logger *zap.Logger
	c      *config.WgSpec
}

// NewWGPlugin creates a new WGPlugin.
func NewWGPlugin(logger *zap.Logger, c *config.WgSpec) *WGPlugin {
	return &WGPlugin{
		logger: logger,
		c:      c,
	}
}

// Name returns the name of the plugin.
func (p *WGPlugin) Name() string {
	return _name
}

// Handle executes the shell command when a message is received.
func (p *WGPlugin) Handle(ctx context.Context, msg *apiv1.HostMessage) {
	switch msg.Type {
	case apiv1.HostMessage_HostUpdateNotification:
		p.handleHostUpdateNotification(ctx, msg)
	case apiv1.HostMessage_HostPunchNotification:
		// TODO 暂时使用这个类型，后续需要修改
		p.handleHostPunchNotification(ctx, msg)
	}
}

// isConcerned checks if the hostname is in the list of concerns.
func isConcerned(concerns []string, hostname string) bool {
	for _, concern := range concerns {
		if concern == hostname {
			return true
		}
	}
	return false
}

func (p *WGPlugin) SetPeerEndpoint(iface string, peer string, endpoint string) error {
	cmd := []string{"wg", "set", iface, "peer", peer, "endpoint", endpoint}
	output, err := run(
		"wg", "set", iface,
		"peer", peer,
		"persistent-keepalive", "10",
		"endpoint", endpoint,
	)
	if err != nil {
		p.logger.Error("failed to execute shell command", zap.Error(err), zap.String("cmd", strings.Join(cmd, " ")), zap.String("output", output))
	} else {
		p.logger.Info("Shell command executed successfully", zap.String("cmd", strings.Join(cmd, " ")), zap.String("output", output))
	}
	return nil
}

func (p *WGPlugin) handleHostUpdateNotification(ctx context.Context, msg *apiv1.HostMessage) {
	// TODO 暂时默认第一个接口
	//msg.Hostname = p.c.Interfaces[0].Publickey
}

func (p *WGPlugin) handleHostPunchNotification(ctx context.Context, msg *apiv1.HostMessage) {
	hostname := msg.Hostname
	externalAddr := utils.NewUDPAddrFromLH4(msg.ExternalAddr).String()

	if err := p.SetPeerEndpoint(p.c.Iface, hostname, externalAddr); err != nil {
		p.logger.Error("Failed to set peer endpoint",
			zap.String("interface", p.c.Iface),
			zap.String("hostname", hostname),
			zap.String("externalAddr", externalAddr),
			zap.Error(err))
	}

	//for _, iface := range p.c.Interfaces {
	//	if isConcerned(iface.Concern, hostname) {
	//		if err := p.SetPeerEndpoint(iface.Iface, hostname, externalAddr); err != nil {
	//			p.logger.Error("Failed to set peer endpoint",
	//				zap.String("interface", iface.Iface),
	//				zap.String("hostname", hostname),
	//				zap.String("externalAddr", externalAddr),
	//				zap.Error(err))
	//		}
	//	}
	//}
}

func run(cmd string, args ...string) (string, error) {
	b, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
