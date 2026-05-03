package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/accounts"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
)

func containerList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	limit := intFlag(opts, "limit")
	if limit == 0 && !boolFlag(opts, "all") {
		limit = 10000
	}
	page, err := containers.List(client, containers.ListOpts{
		Limit:     limit,
		Marker:    flagValue(opts, "marker"),
		EndMarker: flagValue(opts, "end-marker"),
		Prefix:    flagValue(opts, "prefix"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := containers.ExtractInfo(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		row := outputRow{"Name": item.Name}
		if boolFlag(opts, "long") {
			row["Bytes"] = item.Bytes
			row["Count"] = item.Count
		}
		rows = append(rows, row)
	}
	columns := []string{"Name"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Bytes", "Count")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func containerShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("container show requires <container>")
	}
	result := containers.Get(ctx, client, args[0], nil)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	metadata, err := result.ExtractMetadata()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"container", args[0]},
		{"bytes_used", item.BytesUsed},
		{"object_count", item.ObjectCount},
		{"content_type", item.ContentType},
		{"read_acl", compactStrings(item.Read)},
		{"write_acl", compactStrings(item.Write)},
		{"storage_policy", item.StoragePolicy},
		{"versions_location", item.VersionsLocation},
		{"history_location", item.HistoryLocation},
		{"metadata", metadata},
	})
}

func objectList(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("object list requires <container>")
	}
	limit := intFlag(opts, "limit")
	if limit == 0 && !boolFlag(opts, "all") {
		limit = 10000
	}
	page, err := objects.List(client, args[0], objects.ListOpts{
		Limit:     limit,
		Marker:    flagValue(opts, "marker"),
		EndMarker: flagValue(opts, "end-marker"),
		Prefix:    flagValue(opts, "prefix"),
		Delimiter: flagValue(opts, "delimiter"),
	}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := objects.ExtractInfo(page)
	if err != nil {
		return err
	}
	rows := make([]outputRow, 0, len(items))
	for _, item := range items {
		name := item.Name
		if name == "" {
			name = item.Subdir
		}
		row := outputRow{"Name": name}
		if boolFlag(opts, "long") {
			row["Bytes"] = item.Bytes
			row["Hash"] = item.Hash
			row["Content Type"] = item.ContentType
			row["Last Modified"] = item.LastModified
		}
		rows = append(rows, row)
	}
	columns := []string{"Name"}
	if boolFlag(opts, "long") {
		columns = append(columns, "Bytes", "Hash", "Content Type", "Last Modified")
	}
	return renderListOutput(stdout, opts, columns, rows)
}

func objectShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("object show requires <container> <object>")
	}
	result := objects.Get(ctx, client, args[0], args[1], nil)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	metadata, err := result.ExtractMetadata()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"container", args[0]},
		{"object", args[1]},
		{"content_length", item.ContentLength},
		{"content_type", item.ContentType},
		{"etag", item.ETag},
		{"last_modified", item.LastModified},
		{"content_disposition", item.ContentDisposition},
		{"content_encoding", item.ContentEncoding},
		{"object_manifest", item.ObjectManifest},
		{"static_large_object", item.StaticLargeObject},
		{"metadata", metadata},
	})
}

func objectStoreAccountShow(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient) error {
	result := accounts.Get(ctx, client, nil)
	item, err := result.Extract()
	if err != nil {
		return err
	}
	metadata, err := result.ExtractMetadata()
	if err != nil {
		return err
	}
	return renderShowOutput(stdout, opts, []outputField{
		{"bytes_used", item.BytesUsed},
		{"container_count", item.ContainerCount},
		{"object_count", item.ObjectCount},
		{"content_type", item.ContentType},
		{"quota_bytes", item.QuotaBytes},
		{"metadata", metadata},
	})
}
