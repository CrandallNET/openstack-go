package keystoneextras

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
		ID: caddy.ModuleID(cliplugin.NamespaceExtras + ".keystone-extras"),
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (Module) PluginCommands() []cliplugin.Command {
	commands := []string{
		"endpoint group add project",
		"endpoint group create",
		"endpoint group delete",
		"endpoint group list",
		"endpoint group remove project",
		"endpoint group set",
		"endpoint group show",
		"federation domain list",
		"federation project list",
		"federation protocol create",
		"federation protocol delete",
		"federation protocol list",
		"federation protocol set",
		"federation protocol show",
		"identity provider create",
		"identity provider delete",
		"identity provider list",
		"identity provider set",
		"identity provider show",
		"service provider create",
		"service provider delete",
		"service provider list",
		"service provider set",
		"service provider show",
	}
	result := make([]cliplugin.Command, 0, len(commands))
	for _, command := range commands {
		result = append(result, cliplugin.Command{
			Path:        command,
			Group:       "openstack.identity.v3",
			Summary:     "Handle Keystone REST compatibility gaps through the Keystone extras shim.",
			Implemented: true,
		})
	}
	return result
}
