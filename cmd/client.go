package cmd

import (
	"github.com/cossteam/punchline/api/signaling/v1"
	"github.com/cossteam/punchline/pkg/controller"
	"github.com/cossteam/punchline/pkg/ice"
	"github.com/cossteam/punchline/pkg/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
			Name:  "signalServer",
			Usage: "signalServer",
			Value: "",
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

	conn, err := grpc.NewClient(c.SignalServer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	signalingClient := signaling.NewSignalingClient(conn)

	var peers []controller.Runnable
	for _, sub := range c.Subscriptions {
		wrapper, err := ice.NewICEAgentWrapper(logger, signalingClient, c.Hostname, sub.Topic)
		if err != nil {
			return err
		}
		peers = append(peers, wrapper)
	}

	ctrl := controller.NewManager(
		logger.With(zap.String("controller", "manager")),
		peers...,
	//client,
	)
	return ctrl.Start(SetupSignalHandler())
}
