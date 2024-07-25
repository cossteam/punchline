package cmd

import (
	"fmt"
	"github.com/cossteam/punchline/pkg/controller"
	controllerClient "github.com/cossteam/punchline/pkg/controller/client"
	"github.com/cossteam/punchline/pkg/log"
	plugin "github.com/cossteam/punchline/pkg/plugin/client"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"net"
)

func init() {
	App.Commands = append(App.Commands, Client)
}

var Client = &cli.Command{
	Name:  "client",
	Usage: "client",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "config file path",
			Value:   "config.yaml",
		},
		&cli.StringFlag{
			Name:    "loglevel",
			Aliases: []string{"ll"},
			Usage:   "log level (debug info warn error dpanic panic fatal)",
			Value:   "debug",
		},
		&cli.StringFlag{
			Name:  "hostname",
			Usage: "hostname",
			Value: "",
		},
		&cli.StringFlag{
			Name:    "endpointPort",
			Usage:   "endpointPort",
			Aliases: []string{"ep"},
			Value:   "0",
		},
		&cli.StringFlag{
			Name:    "server",
			Usage:   "server",
			Aliases: []string{"srv"},
			Value:   "",
		},
		&cli.StringFlag{
			Name:    "stunServer",
			Usage:   "stunServer",
			Aliases: []string{"ss"},
			Value:   "stun:stun.easyvoip.com:3478",
		},
		&cli.StringFlag{
			Name:    "grpcServer",
			Usage:   "grpcServer",
			Aliases: []string{"gs"},
			Value:   "",
		},
	},
	Action: runClient,
}

func runClient(ctx *cli.Context) error {
	c, err := applyConfig(ctx)
	if err != nil {
		return err
	}

	logger, err := log.SetupLogger(c.Logging.Level)
	if err != nil {
		return err
	}

	raddr, err := net.ResolveUDPAddr("udp", c.Server)
	if err != nil {
		return err
	}

	makeup, err := udp.DialMakeup(raddr.IP, raddr.Port)
	if err != nil {
		return fmt.Errorf("failed to dial makeup: %w", err)
	}

	coordinator := make([]*net.UDPAddr, 0)
	if raddr != nil {
		coordinator = append(coordinator, raddr)
	}

	ps, err := plugin.LoadPlugins(logger, c)
	if err != nil {
		return err
	}

	client := controllerClient.NewClientController(
		logger.With(zap.String("controller", "client")),
		c.Hostname,
		uint32(c.EndpointPort),
		makeup,
		coordinator,
		c,
		controllerClient.WithClientPlugins(ps),
	)

	ctrl := controller.NewManager(
		logger.With(zap.String("controller", "manager")),
		client,
	)
	return ctrl.Start(SetupSignalHandler())
}
