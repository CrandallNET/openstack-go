package cli

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/spf13/cobra"
)

func isCinderExtrasCommand(path string) bool {
	switch path {
	case "block storage resource filter list", "block storage resource filter show":
		return true
	default:
		return false
	}
}

func runCinderExtras(path string, stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		clients, err := newOpenStackClients(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := clients.blockStorageV3()
		if err != nil {
			return err
		}

		switch path {
		case "block storage resource filter list":
			return blockStorageResourceFilterList(cmd.Context(), stdout, opts, client)
		case "block storage resource filter show":
			return blockStorageResourceFilterShow(cmd.Context(), stdout, opts, client, args)
		default:
			return fmt.Errorf("unsupported cinder extras command %q", path)
		}
	}
}

type blockStorageResourceFilterRecord struct {
	Resource string   `json:"resource"`
	Filters  []string `json:"filters"`
}

func blockStorageResourceFilterList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	items, err := listBlockStorageResourceFilters(ctx, client, "")
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, outputRow{
			"Resource": item.Resource,
			"Filters":  blockStorageResourceFiltersValue(item.Filters),
		})
	}
	return renderListOutput(stdout, opts, []string{"Resource", "Filters"}, rows)
}

func blockStorageResourceFilterShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("block storage resource filter show requires <resource>")
	}
	items, err := listBlockStorageResourceFilters(ctx, client, args[0])
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("No resource filter with a name of {parsed_args.resource}' exists.")
	}
	item := items[0]
	return renderShowOutput(stdout, opts, []outputField{
		{"Resource", item.Resource},
		{"Filters", blockStorageResourceFiltersValue(item.Filters)},
	})
}

func blockStorageResourceFiltersValue(filters []string) tableValue {
	value := append([]string(nil), filters...)
	tableFilters := append([]string(nil), filters...)
	sort.Strings(tableFilters)
	return tableValue{Value: value, Table: strings.Join(tableFilters, ", "), Pretty: tableFilters}
}

func listBlockStorageResourceFilters(ctx context.Context, client *gophercloud.ServiceClient, resource string) ([]blockStorageResourceFilterRecord, error) {
	client, err := blockStorageClientWithMinimumMicroversion(ctx, client, "3.33")
	if err != nil {
		return nil, err
	}
	requestURL := client.ServiceURL("resource_filters")
	if resource != "" {
		query := url.Values{}
		query.Set("resource", resource)
		requestURL += "?" + query.Encode()
	}
	var response struct {
		ResourceFilters []blockStorageResourceFilterRecord `json:"resource_filters"`
	}
	resp, err := client.Get(ctx, requestURL, &response, nil)
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return response.ResourceFilters, nil
}
