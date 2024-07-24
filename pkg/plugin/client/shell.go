package plugin

import (
	"context"
	apiv1 "github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/pkg/utils"
	"go.uber.org/zap"
	"os/exec"
	"strings"
)

// ShellCommandPlugin is a plugin that executes a shell command when a message is received.
type ShellCommandPlugin struct {
	logger *zap.Logger
	name   string
	iface  string
}

// NewShellCommandPlugin creates a new ShellCommandPlugin.
func NewShellCommandPlugin(logger *zap.Logger, name string) *ShellCommandPlugin {
	return &ShellCommandPlugin{
		logger: logger,
		name:   name,
		iface:  "wg0",
	}
}

// Name returns the name of the plugin.
func (p *ShellCommandPlugin) Name() string {
	return p.name
}

// Handle executes the shell command when a message is received.
func (p *ShellCommandPlugin) Handle(ctx context.Context, msg *apiv1.HostMessage) {
	// Execute the shell command
	_ = p.SetPeerEndpoint(p.iface, msg.Hostname, utils.NewUDPAddrFromLH4(msg.ExternalAddr).String())
}

func (p *ShellCommandPlugin) SetPeerEndpoint(iface string, peer string, endpoint string) error {
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
		p.logger.Info("shell command executed successfully", zap.String("cmd", strings.Join(cmd, " ")), zap.String("output", output))
	}
	return nil
}

func run(cmd string, args ...string) (string, error) {
	b, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
