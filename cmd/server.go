package cmd

import (
	"fmt"
	"github.com/cossteam/punchline/config"
	"github.com/cossteam/punchline/pkg/controller"
	"github.com/cossteam/punchline/pkg/log"
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
			Name:    "udp_port",
			Usage:   "udp_port",
			Aliases: []string{"up"},
			Value:   "6976",
		},
		&cli.StringFlag{
			Name:  "hostname",
			Usage: "hostname",
			Value: "server-1",
		},
		&cli.StringFlag{
			Name:    "grpc_port",
			Usage:   "grpc_port",
			Aliases: []string{"gp"},
			Value:   "7777",
		},
	},
	Action: runServer,
}

func runServer(ctx *cli.Context) error {
	c, err := applyConfig(ctx)
	if err != nil {
		return err
	}

	uaddr := fmt.Sprintf("%s:%d", "0.0.0.0", c.UdpPort)

	logger, err := log.SetupLogger(c.Loglevel)
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

	srv := controller.NewServer(logger.With(zap.String("controller", "server")), uint32(c.GrpcPort), c.Hostname, outside, c)

	//var runnables []apiv1.Runnable
	//runnables = append(runnables, srv)

	return srv.Start(SetupSignalHandler())
}

func applyConfig(ctx *cli.Context) (config2 *config.Config, err error) {
	cfg := &config.Config{}
	if ctx.String("config") != "" {
		cfg, err = config.Load(ctx.String("config"))
		if err != nil {
			return nil, err
		}
	}

	// Apply command line flags, overriding configuration file values
	udpPort := ctx.Uint("udp_port")
	if udpPort != 0 {
		cfg.UdpPort = udpPort
	}

	grpcPort := ctx.Uint("grpc_port")
	if grpcPort != 0 {
		cfg.GrpcPort = grpcPort
	}

	endpointPort := ctx.Uint("endpoint_port")
	if endpointPort != 0 {
		cfg.EndpointPort = endpointPort
	}

	logLevel := ctx.String("loglevel")
	if logLevel != "" {
		cfg.Loglevel = logLevel
	}

	hostname := ctx.String("hostname")
	if hostname != "" {
		cfg.Hostname = hostname
	}

	server := ctx.String("server")
	if server != "" {
		cfg.Server = server
	}

	return cfg, nil
}
