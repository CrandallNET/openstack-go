package cli

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

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

func containerCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("container create requires <container-name>")
	}
	createOpts := containers.CreateOpts{
		StoragePolicy: flagValue(opts, "storage-policy"),
	}
	if boolFlag(opts, "public") {
		createOpts.ContainerRead = ".r:*,.rlistings"
	}
	rows := make([]outputRow, 0, len(args))
	account := objectStorageAccountName(client)
	for _, name := range args {
		result := containers.Create(ctx, client, name, createOpts)
		if result.Err != nil {
			return result.Err
		}
		rows = append(rows, outputRow{
			"account":    account,
			"container":  name,
			"x-trans-id": result.Header.Get("X-Trans-Id"),
		})
	}
	return renderListOutput(stdout, opts, []string{"account", "container", "x-trans-id"}, rows)
}

func containerDelete(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("container delete requires <container>")
	}
	for _, container := range args {
		if boolFlag(opts, "recursive") {
			if err := deleteContainerObjects(ctx, client, container); err != nil {
				return err
			}
		}
		if result := containers.Delete(ctx, client, container); result.Err != nil {
			return result.Err
		}
	}
	return nil
}

func containerSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("container set requires <container>")
	}
	properties, err := containerProperties(opts)
	if err != nil {
		return err
	}
	if len(properties) == 0 {
		return fmt.Errorf("container set requires --property")
	}
	if result := containers.Update(ctx, client, args[0], containers.UpdateOpts{Metadata: properties}); result.Err != nil {
		return result.Err
	}
	return nil
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
	fields := []outputField{
		{"account", objectStorageAccountName(client)},
		{"bytes_used", fmt.Sprintf("%d", item.BytesUsed)},
		{"container", args[0]},
		{"object_count", fmt.Sprintf("%d", item.ObjectCount)},
	}
	if len(metadata) > 0 {
		fields = append(fields, outputField{"properties", metadata})
	}
	if item.StoragePolicy != "" {
		fields = append(fields, outputField{"storage_policy", item.StoragePolicy})
	}
	return renderShowOutput(stdout, opts, fields)
}

func containerUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("container unset requires <container>")
	}
	properties := flagValues(opts, "property")
	if len(properties) == 0 {
		return fmt.Errorf("container unset requires --property")
	}
	if result := containers.Update(ctx, client, args[0], containers.UpdateOpts{RemoveMetadata: properties}); result.Err != nil {
		return result.Err
	}
	return nil
}

func deleteContainerObjects(ctx context.Context, client *gophercloud.ServiceClient, container string) error {
	page, err := objects.List(client, container, objects.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	items, err := objects.ExtractInfo(page)
	if err != nil {
		return err
	}
	for _, item := range items {
		name := item.Name
		if name == "" {
			name = item.Subdir
		}
		if name == "" {
			continue
		}
		if result := objects.Delete(ctx, client, container, name, nil); result.Err != nil {
			return result.Err
		}
	}
	return nil
}

func containerProperties(opts *Options) (map[string]string, error) {
	properties := map[string]string{}
	for _, property := range flagValues(opts, "property") {
		key, value, ok := strings.Cut(property, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid property %q, expected <key=value>", property)
		}
		properties[key] = value
	}
	return properties, nil
}

func objectStorageAccountName(client *gophercloud.ServiceClient) string {
	parsed, err := url.Parse(client.Endpoint)
	if err != nil {
		return ""
	}
	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
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

func objectCreate(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("object create requires <container> <filename>")
	}
	container := args[0]
	files := args[1:]
	if flagValue(opts, "name") != "" && len(files) > 1 {
		return fmt.Errorf("Attempting to upload multiple objects and using --name is not permitted")
	}
	rows := make([]outputRow, 0, len(files))
	for _, filename := range files {
		file, err := os.Open(expandUserPath(filename))
		if err != nil {
			return err
		}
		objectName := filename
		if override := flagValue(opts, "name"); override != "" {
			objectName = override
		}
		result := objects.Create(ctx, client, container, objectName, objects.CreateOpts{Content: file})
		_ = file.Close()
		if result.Err != nil {
			return result.Err
		}
		rows = append(rows, outputRow{
			"object":    objectName,
			"container": container,
			"etag":      result.Header.Get("Etag"),
		})
	}
	return renderListOutput(stdout, opts, []string{"object", "container", "etag"}, rows)
}

func objectDelete(ctx context.Context, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("object delete requires <container> <object>")
	}
	container := args[0]
	for _, objectName := range args[1:] {
		if result := objects.Delete(ctx, client, container, objectName, nil); result.Err != nil {
			return result.Err
		}
	}
	return nil
}

func objectSave(ctx context.Context, stdout io.Writer, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("object save requires <container> <object>")
	}
	result := objects.Download(ctx, client, args[0], args[1], nil)
	content, err := result.ExtractContent()
	if err != nil {
		return err
	}
	destination := flagValue(opts, "file")
	if destination == "" {
		destination = args[1]
	}
	if destination == "-" {
		_, err := stdout.Write(content)
		return err
	}
	return os.WriteFile(expandUserPath(destination), content, 0600)
}

func objectSet(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("object set requires <container> <object>")
	}
	properties, err := objectProperties(opts)
	if err != nil {
		return err
	}
	if len(properties) == 0 {
		return fmt.Errorf("object set requires --property")
	}
	if result := objects.Update(ctx, client, args[0], args[1], objects.UpdateOpts{Metadata: properties}); result.Err != nil {
		return result.Err
	}
	return nil
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
	fields := []outputField{
		{"account", objectStorageAccountName(client)},
		{"container", args[0]},
		{"content-length", fmt.Sprintf("%d", item.ContentLength)},
		{"content-type", item.ContentType},
		{"etag", item.ETag},
	}
	if !item.LastModified.IsZero() {
		fields = append(fields, outputField{"last-modified", item.LastModified.Format(time.RFC1123)})
	}
	fields = append(fields, outputField{"object", args[1]})
	if len(metadata) > 0 {
		fields = append(fields, outputField{"properties", metadata})
	}
	return renderShowOutput(stdout, opts, fields)
}

func objectUnset(ctx context.Context, opts *Options, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("object unset requires <container> <object>")
	}
	properties := flagValues(opts, "property")
	if len(properties) == 0 {
		return fmt.Errorf("object unset requires --property")
	}
	if result := objects.Update(ctx, client, args[0], args[1], objects.UpdateOpts{RemoveMetadata: properties}); result.Err != nil {
		return result.Err
	}
	return nil
}

func objectProperties(opts *Options) (map[string]string, error) {
	properties := map[string]string{}
	for _, property := range flagValues(opts, "property") {
		key, value, ok := strings.Cut(property, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid property %q, expected <key=value>", property)
		}
		properties[key] = value
	}
	return properties, nil
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
