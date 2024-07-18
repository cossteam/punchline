package api

import "context"

type EncWriter interface {
	//SendTo(t header.MessageType, st header.MessageSubType, vpnIp api.VpnIp, p, out []byte)
}

type Runnable interface {
	// Start starts running the component.  The component will stop running
	// when the context is closed. Start blocks until the context is closed or
	// an error occurs.
	Start(context.Context) error
}
