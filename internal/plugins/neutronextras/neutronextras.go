package neutronextras

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
		ID: caddy.ModuleID(cliplugin.NamespaceExtras + ".neutron-extras"),
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (Module) PluginCommands() []cliplugin.Command {
	commands := []string{
		"default security group rule create",
		"default security group rule delete",
		"default security group rule list",
		"default security group rule show",
		"local ip association create",
		"local ip association delete",
		"local ip association list",
		"local ip create",
		"local ip delete",
		"local ip list",
		"local ip set",
		"local ip show",
		"network agent add network",
		"network agent add router",
		"network agent delete",
		"network agent remove network",
		"network agent remove router",
		"network agent set",
		"network auto allocated topology create",
		"network auto allocated topology delete",
		"network flavor add profile",
		"network flavor create",
		"network flavor delete",
		"network flavor list",
		"network flavor profile create",
		"network flavor profile delete",
		"network flavor profile list",
		"network flavor profile set",
		"network flavor profile show",
		"network flavor remove profile",
		"network flavor set",
		"network flavor show",
		"network l3 conntrack helper create",
		"network l3 conntrack helper delete",
		"network l3 conntrack helper list",
		"network l3 conntrack helper set",
		"network l3 conntrack helper show",
		"network meter create",
		"network meter delete",
		"network meter list",
		"network meter rule create",
		"network meter rule delete",
		"network meter rule list",
		"network meter rule show",
		"network meter show",
		"network segment range create",
		"network segment range delete",
		"network segment range list",
		"network segment range set",
		"network segment range show",
		"router ndp proxy create",
		"router ndp proxy delete",
		"router ndp proxy list",
		"router ndp proxy set",
		"router ndp proxy show",
		"tap flow create",
		"tap flow delete",
		"tap flow list",
		"tap flow show",
		"tap flow update",
		"tap mirror create",
		"tap mirror delete",
		"tap mirror list",
		"tap mirror show",
		"tap mirror update",
		"tap service create",
		"tap service delete",
		"tap service list",
		"tap service show",
		"tap service update",
	}
	items := make([]cliplugin.Command, 0, len(commands))
	for _, command := range commands {
		items = append(items, cliplugin.Command{
			Path:        command,
			Group:       "openstack.network.v2",
			Summary:     "Run a Neutron extension command through the neutron-extras shim.",
			Implemented: true,
		})
	}
	return items
}
