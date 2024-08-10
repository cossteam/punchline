package cmd

import (
	"context"
	"github.com/cossteam/punchline/config"
	"github.com/urfave/cli/v2"
	"os"
	"os/signal"
	"syscall"
)

var App = &cli.App{
	Name:     "nexus",
	Usage:    "nexus",
	Version:  "0.0.1",
	Commands: []*cli.Command{},
}

var onlyOneSignalHandler = make(chan struct{})

// SetupSignalHandler registers for SIGTERM and SIGINT. A context is returned
// which is canceled on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() context.Context {
	close(onlyOneSignalHandler) // panics when called twice

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		cancel()
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return ctx
}

var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT}

func applyConfig(ctx *cli.Context) (config2 *config.Config, err error) {
	cfg := &config.Config{}
	if ctx.String("config") != "" {
		cfg, err = config.Load(ctx.String("config"))
		if err != nil {
			return nil, err
		}
	}

	// Apply command line flags, overriding configuration file values
	//udpPort := ctx.Uint("udpPort")
	//if udpPort != 0 {
	//	cfg.UdpPort = udpPort
	//}

	grpcServer := ctx.String("grpcServer")
	if grpcServer != "" {
		cfg.SignalServer = grpcServer
	}

	endpointPort := ctx.Uint("endpointPort")
	if endpointPort != 0 {
		cfg.EndpointPort = endpointPort
	}

	logLevel := ctx.String("loglevel")
	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	hostname := ctx.String("hostname")
	if hostname != "" {
		cfg.Hostname = hostname
	}

	server := ctx.String("server")
	if server != "" {
		cfg.Server = server
	}

	stunServer := ctx.String("stunServer")
	if stunServer != "" {
		cfg.StunServer = stunServer
	}

	return cfg, nil
}
