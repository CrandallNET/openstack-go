package novaextras

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
		ID: caddy.ModuleID(cliplugin.NamespaceExtras + ".nova-extras"),
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (Module) PluginCommands() []cliplugin.Command {
	return []cliplugin.Command{
		{
			Path:        "server ssh",
			Group:       "openstack.compute.v2",
			Summary:     "SSH to a server through the pure Go Nova extras SSH client.",
			Implemented: true,
		},
	}
}
