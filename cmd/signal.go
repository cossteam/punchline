package cmd

import (
	"github.com/cossteam/punchline/pkg/controller"
	"github.com/cossteam/punchline/pkg/controller/signaling"
	"github.com/cossteam/punchline/pkg/log"
	plugin "github.com/cossteam/punchline/pkg/plugin/client"
	"github.com/urfave/cli/v2"
)

func init() {
	App.Commands = append(App.Commands, Signal)
}

var Signal = &cli.Command{
	Name:  "signal",
	Usage: "signal",
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
			Name:  "addr",
			Usage: "addr",
			Value: "0.0.0.0:7777",
		},
	},
	Action: runSignal,
}

func runSignal(ctx *cli.Context) error {
	c, err := applyConfig(ctx)
	if err != nil {
		return err
	}

	addr := ctx.String("addr")

	logger, err := log.SetupLogger(c.Logging.Level)
	if err != nil {
		return err
	}

	_, err = plugin.LoadPlugins(logger, c)
	if err != nil {
		return err
	}

	srv := signaling.NewSignalingController(addr, logger)
	ctrl := controller.NewManager(logger, srv)
	return ctrl.Start(SetupSignalHandler())
}
