package plugin

import (
	"context"
	"fmt"
	apiv1 "github.com/cossteam/punchline/api/v1"
	"github.com/cossteam/punchline/config"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	Name() string
	Handle(ctx context.Context, msg *apiv1.HostMessage)
}

// LoadPlugins loads plugins based on the configuration
func LoadPlugins(logger *zap.Logger, cfg *config.Config) ([]Plugin, error) {
	var plugins []Plugin
	for _, pluginConfig := range cfg.Plugins {
		switch pluginConfig.Name {
		case "wg":
			var spec config.WgSpec
			if err := mapstructure.Decode(pluginConfig.Spec, &spec); err != nil {
				return nil, fmt.Errorf("failed to decode wg plugin spec: %v", err)
			}
			plugins = append(plugins, NewWGPlugin(logger.With(zap.String("plugin", pluginConfig.Name)), &spec))
		default:
			logger.Warn("unknown plugin", zap.String("plugin", pluginConfig.Name))
		}
	}
	return plugins, nil
}

// ExamplePlugin is an example implementation of the Plugin interface.
type ExamplePlugin struct {
	// Add fields if needed
}

func (p *ExamplePlugin) Name() string {
	return "ExamplePlugin"
}

func (p *ExamplePlugin) Handle(ctx context.Context, msg *apiv1.HostMessage) {
	// Handle the message
}
