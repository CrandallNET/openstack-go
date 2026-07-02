package local

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/crandallnet/openstack-go/internal/cliplugin"
)

func init() {
	caddy.RegisterModule(Module{})
}

type Module struct{}

func (Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: caddy.ModuleID(cliplugin.NamespaceCore + ".local"),
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (Module) PluginCommands() []cliplugin.Command {
	return []cliplugin.Command{
		{
			Path:        "command list",
			Group:       "openstack.cli",
			Summary:     "List recognized commands from the embedded OSC catalog.",
			Implemented: true,
		},
		{
			Path:        "module list",
			Group:       "openstack.cli",
			Summary:     "List golang-osc command provider modules.",
			Implemented: true,
		},
	}
}
