package cli

import (
	"context"
	"os"
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

func (clients *openStackClients) computeV2() (*gophercloud.ServiceClient, error) {
	return openstack.NewComputeV2(clients.Provider, clients.EndpointOpts)
}

func (clients *openStackClients) imageV2() (*gophercloud.ServiceClient, error) {
	return openstack.NewImageV2(clients.Provider, clients.EndpointOpts)
}

func (clients *openStackClients) networkV2() (*gophercloud.ServiceClient, error) {
	return openstack.NewNetworkV2(clients.Provider, clients.EndpointOpts)
}

func (clients *openStackClients) blockStorageV3() (*gophercloud.ServiceClient, error) {
	return openstack.NewBlockStorageV3(clients.Provider, clients.EndpointOpts)
}

func (clients *openStackClients) objectStorageV1() (*gophercloud.ServiceClient, error) {
	return openstack.NewObjectStorageV1(clients.Provider, clients.EndpointOpts)
}

func (clients *openStackClients) placementV1() (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewPlacementV1(clients.Provider, clients.EndpointOpts)
	if err != nil {
		return nil, err
	}
	client.Microversion = os.Getenv("OS_PLACEMENT_API_VERSION")
	if client.Microversion == "" {
		client.Microversion = "1.39"
	}
	return client, nil
}
