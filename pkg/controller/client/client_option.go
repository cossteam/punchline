package controller

import plugin "github.com/cossteam/punchline/pkg/plugin/client"

func WithClientPlugin(plugin plugin.Plugin) ClientOption {
	return func(cc *clientController) {
		cc.plugins = append(cc.plugins, plugin)
	}
}

func WithClientPlugins(plugins []plugin.Plugin) ClientOption {
	return func(cc *clientController) {
		cc.plugins = append(cc.plugins, plugins...)
	}
}
