package cmd

import (
	"github.com/cossteam/punchline/pkg/controller"
	"github.com/cossteam/punchline/pkg/ice"
	"github.com/cossteam/punchline/pkg/log"
	"github.com/cossteam/punchline/pkg/signal"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
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
			Value:   "",
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
			Name:  "signalServer",
			Usage: "signalServer",
			Value: "",
		},
		&cli.StringSliceFlag{
			Name:    "stunServer",
			Aliases: []string{"ss"},
			Usage:   "List of names",
			Value:   cli.NewStringSlice("stun:stun3.l.google.com:19302", "stun:stun.cunicu.li:3478", "stun:stun.easyvoip.com:3478"),
		},
		&cli.StringSliceFlag{
			Name:    "subscriptions",
			Aliases: []string{"s"},
			Usage:   "Subscribed Hosts",
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

	//raddr, err := net.ResolveUDPAddr("udp", c.Server)
	//if err != nil {
	//	return err
	//}
	//
	//makeup, err := udp.DialMakeup(raddr.IP, raddr.Port)
	//if err != nil {
	//	return fmt.Errorf("failed to dial makeup: %w", err)
	//}

	//coordinator := make([]*net.UDPAddr, 0)
	//if raddr != nil {
	//	coordinator = append(coordinator, raddr)
	//}

	//ps, err := plugin.LoadPlugins(logger, c)
	//if err != nil {
	//	return err
	//}

	//client := controllerClient.NewClientController(
	//	logger.With(zap.String("controller", "client")),
	//	c.Hostname,
	//	uint32(c.EndpointPort),
	//	makeup,
	//	coordinator,
	//	c,
	//	controllerClient.WithClientPlugins(ps),
	//)

	signalingClient, err := signal.NewClient(c.SignalServer, signal.WithClientName(c.Hostname))
	if err != nil {
		return err
	}

	var peers []controller.Runnable
	for _, sub := range c.Subscriptions {
		wrapper, err := ice.NewICEAgentWrapper(logger, signalingClient, c.StunServer, c.Hostname, sub.Topic)
		if err != nil {
			return err
		}
		peers = append(peers, wrapper)
	}

	ctrl := controller.NewManager(
		logger.With(zap.String("controller", "manager")),
		peers...,
	)
	return ctrl.Start(SetupSignalHandler())
}
