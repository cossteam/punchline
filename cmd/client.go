package cmd

import (
	"fmt"
	"github.com/cossteam/punchline/pkg/controller"
	"github.com/cossteam/punchline/pkg/log"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/urfave/cli/v2"
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
			Name:    "loglevel",
			Aliases: []string{"ll"},
			Usage:   "log level (debug info warn error dpanic panic fatal)",
			Value:   "debug",
		},
		&cli.StringFlag{
			Name:  "hostname",
			Usage: "hostname",
			Value: "client-1",
		},
		&cli.StringFlag{
			Name:    "port",
			Usage:   "port",
			Aliases: []string{"p"},
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
	logLevel := ctx.String("loglevel")
	hostname := ctx.String("hostname")
	listenPort := ctx.Int("port")
	server := ctx.String("server")

	logger, err := log.SetupLogger(logLevel)
	if err != nil {
		return err
	}

	raddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return err
	}

	makeup, err := udp.DialMakeup(raddr.IP)
	if err != nil {
		return fmt.Errorf("failed to dial makeup: %w", err)
	}

	coordinator := make([]*net.UDPAddr, 0)
	if raddr != nil {
		coordinator = append(coordinator, raddr)
	}

	client := controller.NewClient(logger, uint32(listenPort), hostname, makeup, coordinator)

	return client.Start(ctx.Context)
}
