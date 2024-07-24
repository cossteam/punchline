package plugin

import (
	"context"
	apiv1 "github.com/cossteam/punchline/api/v1"
)

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	Name() string
	Handle(ctx context.Context, msg *apiv1.HostMessage)
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
