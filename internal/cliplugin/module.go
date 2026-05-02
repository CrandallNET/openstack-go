package cliplugin

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
)

const (
	NamespaceCore    = "openstack.commands.core"
	NamespacePlugins = "openstack.commands.plugins"
	NamespaceExtras  = "openstack.commands.extras"
)

type Command struct {
	Path        string
	Group       string
	Summary     string
	Implemented bool
}

type CommandProvider interface {
	caddy.Module
	PluginCommands() []Command
}

func Providers(namespace string) ([]CommandProvider, error) {
	modules := caddy.GetModules(namespace)
	providers := make([]CommandProvider, 0, len(modules))
	for _, module := range modules {
		instance := module.New()
		provider, ok := instance.(CommandProvider)
		if !ok {
			return nil, fmt.Errorf("module %s does not implement cliplugin.CommandProvider", module.ID)
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

func ModuleIDs(namespace string) []string {
	modules := caddy.GetModules(namespace)
	ids := make([]string, 0, len(modules))
	for _, module := range modules {
		ids = append(ids, string(module.ID))
	}
	return ids
}
