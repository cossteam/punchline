package cmd

import (
	"fmt"
	"github.com/cossteam/punchline/pkg/controller"
	"github.com/cossteam/punchline/pkg/log"
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
			Name:    "endpoint_port",
			Usage:   "endpoint_port",
			Aliases: []string{"ep"},
			Value:   "0",
		},
		&cli.StringFlag{
			Name:    "server",
			Usage:   "server",
			Aliases: []string{"srv"},
			Value:   "127.0.0.1:6976",
		},
	},
	Action: runClient,
}

func runClient(ctx *cli.Context) error {
	c, err := applyConfig(ctx)
	if err != nil {
		return err
	}

	logger, err := log.SetupLogger(c.Loglevel)
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

	client := controller.NewClient(logger.With(zap.String("controller", "client")), uint32(c.EndpointPort), c.Hostname, makeup, coordinator, c)

	return client.Start(SetupSignalHandler())
}
