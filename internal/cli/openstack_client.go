package cli

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
)

type openStackClients struct {
	Provider     *gophercloud.ProviderClient
	EndpointOpts gophercloud.EndpointOpts
}

func newOpenStackClients(ctx context.Context, opts *Options) (*openStackClients, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	authOptions, endpointOptions, tlsConfig, err := resolveAuthOptions(opts)
	if err != nil {
		return nil, err
	}

	provider, err := config.NewProviderClient(ctx, authOptions, config.WithTLSConfig(tlsConfig))
	if err != nil {
		return nil, err
	}
	return &openStackClients{Provider: provider, EndpointOpts: endpointOptions}, nil
}

func (clients *openStackClients) identityV3() (*gophercloud.ServiceClient, error) {
	return openstack.NewIdentityV3(clients.Provider, clients.EndpointOpts)
}
