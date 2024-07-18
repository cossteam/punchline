package cmd

import (
	"github.com/cossteam/punchline/pkg/controller"
	"github.com/cossteam/punchline/pkg/log"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/urfave/cli/v2"
	"net"
)

func init() {
	App.Commands = append(App.Commands, Server)
}

var Server = &cli.Command{
	Name:  "server",
	Usage: "server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "loglevel",
			Aliases: []string{"ll"},
			Usage:   "log level (debug info warn error dpanic panic fatal)",
			Value:   "debug",
		},
		&cli.StringFlag{
			Name:  "addr",
			Usage: "addr",
			Value: ":6976",
		},
		&cli.StringFlag{
			Name:  "hostname",
			Usage: "hostname",
			Value: "server-1",
		},
	},
	Action: runServer,
}

func runServer(ctx *cli.Context) error {
	addr := ctx.String("addr")
	logLevel := ctx.String("loglevel")
	hostname := ctx.String("hostname")

	logger, err := log.SetupLogger(logLevel)
	if err != nil {
		return err
	}

	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	outside, err := udp.NewGenericListener(logger, raddr.IP, raddr.Port)
	if err != nil {
		return err
	}

	srv := controller.NewServer(logger, uint32(raddr.Port), hostname, outside)

	//var runnables []apiv1.Runnable
	//runnables = append(runnables, srv)

	return srv.Start(ctx.Context)
}
