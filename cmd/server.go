package cmd

import (
	"github.com/cossteam/punchline/pkg/controller"
	controllersrv "github.com/cossteam/punchline/pkg/controller/server"
	"github.com/cossteam/punchline/pkg/log"
	plugin "github.com/cossteam/punchline/pkg/plugin/client"
	"github.com/cossteam/punchline/pkg/transport/udp"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
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
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "config file path",
			Value:   "",
		},
		&cli.StringFlag{
			Name:    "loglevel",
			Aliases: []string{"ll"},
			Usage:   "log level (debug info warn error dpanic panic fatal)",
			Value:   "debug",
		},
		&cli.StringFlag{
			Name:    "server",
			Usage:   "server",
			Aliases: []string{"srv"},
			Value:   "0.0.0.0:6976",
		},
		&cli.StringFlag{
			Name:  "hostname",
			Usage: "hostname",
			Value: "",
		},
		&cli.StringFlag{
			Name:    "grpcServer",
			Usage:   "grpcServer",
			Aliases: []string{"gs"},
			Value:   "0.0.0.0:7777",
		},
	},
	Action: runServer,
}

func runServer(ctx *cli.Context) error {
	c, err := applyConfig(ctx)
	if err != nil {
		return err
	}

	uaddr := c.Server

	logger, err := log.SetupLogger(c.Logging.Level)
	if err != nil {
		return err
	}

	raddr, err := net.ResolveUDPAddr("udp", uaddr)
	if err != nil {
		return err
	}

	outside, err := udp.NewGenericListener(logger, raddr.IP, raddr.Port)
	if err != nil {
		return err
	}

	_, err = plugin.LoadPlugins(logger, c)
	if err != nil {
		return err
	}

	srv := controllersrv.NewServerController(
		logger.With(zap.String("controller", "server")),
		outside,
		c,
	)

	ctrl := controller.NewManager(
		logger.With(zap.String("controller", "manager")),
		srv,
	)
	return ctrl.Start(SetupSignalHandler())
}
