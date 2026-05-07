package cinderextras

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/crandallnet/golang-osc/internal/cliplugin"
)

func init() {
	caddy.RegisterModule(Module{})
}

type Module struct{}

func (Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: caddy.ModuleID(cliplugin.NamespaceExtras + ".cinder-extras"),
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (Module) PluginCommands() []cliplugin.Command {
	return []cliplugin.Command{
		{
			Path:        "block storage resource filter list",
			Group:       "openstack.volume.v3",
			Summary:     "List block storage resource filters through the Cinder extras shim.",
			Implemented: true,
		},
		{
			Path:        "block storage resource filter show",
			Group:       "openstack.volume.v3",
			Summary:     "Show block storage resource filters through the Cinder extras shim.",
			Implemented: true,
		},
	}
}
