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
	AuthOptions  gophercloud.AuthOptions
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
	return &openStackClients{Provider: provider, EndpointOpts: endpointOptions, AuthOptions: authOptions}, nil
}

func (clients *openStackClients) identityV3() (*gophercloud.ServiceClient, error) {
	return openstack.NewIdentityV3(clients.Provider, clients.EndpointOpts)
}

func (clients *openStackClients) computeV2() (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewComputeV2(clients.Provider, clients.EndpointOpts)
	if err != nil {
		return nil, err
	}
	client.Microversion = os.Getenv("OS_COMPUTE_API_VERSION")
	if client.Microversion == "" {
		client.Microversion = "2.53"
	}
	return client, nil
}

func (clients *openStackClients) imageV2() (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewImageV2(clients.Provider, clients.EndpointOpts)
	if err != nil {
		return nil, err
	}
	client.Microversion = os.Getenv("OS_IMAGE_API_VERSION")
	return client, nil
}

func (clients *openStackClients) networkV2() (*gophercloud.ServiceClient, error) {
	return openstack.NewNetworkV2(clients.Provider, clients.EndpointOpts)
}

func (clients *openStackClients) blockStorageV3() (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewBlockStorageV3(clients.Provider, clients.EndpointOpts)
	if err != nil {
		return nil, err
	}
	client.Microversion = os.Getenv("OS_VOLUME_API_VERSION")
	return client, nil
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
		client.Microversion = "1.29"
	}
	return client, nil
}
