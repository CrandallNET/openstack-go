package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/spf13/cobra"
)

type tokenIssueRow struct {
	Expires   string `json:"expires"`
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
}

var issueToken = issueTokenWithGophercloud

func runTokenIssue(stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		row, err := issueToken(cmd.Context(), opts)
		if err != nil {
			return err
		}
		return renderTokenIssue(stdout, opts, row)
	}
}

func issueTokenWithGophercloud(ctx context.Context, opts *Options) (tokenIssueRow, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	authOptions, _, tlsConfig, err := resolveAuthOptions(opts)
	if err != nil {
		return tokenIssueRow{}, err
	}

	provider, err := config.NewProviderClient(ctx, authOptions, config.WithTLSConfig(tlsConfig))
	if err != nil {
		return tokenIssueRow{}, err
	}

	authResult := provider.GetAuthResult()
	if authResult == nil {
		return tokenIssueRow{ID: provider.Token()}, nil
	}

	switch result := authResult.(type) {
	case tokens3.CreateResult:
		return tokenIssueRowFromV3(result)
	case tokens3.GetResult:
		return tokenIssueRowFromV3(result)
	default:
		tokenID, err := authResult.ExtractTokenID()
		if err != nil {
			return tokenIssueRow{}, err
		}
		return tokenIssueRow{ID: tokenID}, nil
	}
}

func resolveAuthOptions(opts *Options) (gophercloud.AuthOptions, gophercloud.EndpointOpts, *tls.Config, error) {
	parseOptions := cloudParseOptions(opts)
	if opts.Cloud != "" || os.Getenv("OS_CLOUD") != "" {
		authOptions, endpointOptions, tlsConfig, err := clouds.Parse(parseOptions...)
		return authOptions, endpointOptions, tlsConfig, err
	}

	authOptions, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return gophercloud.AuthOptions{}, gophercloud.EndpointOpts{}, nil, err
	}
	applyDirectAuthOverrides(&authOptions, opts)
	return authOptions, endpointOptionsFromOptions(opts), nil, nil
}

func cloudParseOptions(opts *Options) []clouds.ParseOption {
	var parseOptions []clouds.ParseOption
	if opts.Cloud != "" {
		parseOptions = append(parseOptions, clouds.WithCloudName(opts.Cloud))
	}
	if opts.AuthURL != "" {
		parseOptions = append(parseOptions, clouds.WithIdentityEndpoint(opts.AuthURL))
	}
	if opts.ProjectID != "" {
		parseOptions = append(parseOptions, clouds.WithProjectID(opts.ProjectID))
	}
	if opts.ProjectName != "" {
		parseOptions = append(parseOptions, clouds.WithProjectName(opts.ProjectName))
	}
	if opts.Username != "" {
		parseOptions = append(parseOptions, clouds.WithUsername(opts.Username))
	}
	if opts.UserID != "" {
		parseOptions = append(parseOptions, clouds.WithUserID(opts.UserID))
	}
	if opts.Password != "" {
		parseOptions = append(parseOptions, clouds.WithPassword(opts.Password))
	}
	if opts.Token != "" {
		parseOptions = append(parseOptions, clouds.WithToken(opts.Token))
	}
	if opts.RegionName != "" {
		parseOptions = append(parseOptions, clouds.WithRegion(opts.RegionName))
	}
	if opts.Interface != "" {
		parseOptions = append(parseOptions, clouds.WithEndpointType(opts.Interface))
	}
	if opts.Insecure {
		parseOptions = append(parseOptions, clouds.WithInsecure(true))
	}
	if opts.ApplicationCredentialID != "" {
		parseOptions = append(parseOptions, clouds.WithApplicationCredentialID(opts.ApplicationCredentialID))
	}
	if opts.ApplicationCredentialName != "" {
		parseOptions = append(parseOptions, clouds.WithApplicationCredentialName(opts.ApplicationCredentialName))
	}
	if opts.ApplicationCredentialSecret != "" {
		parseOptions = append(parseOptions, clouds.WithApplicationCredentialSecret(opts.ApplicationCredentialSecret))
	}
	return parseOptions
}

func applyDirectAuthOverrides(authOptions *gophercloud.AuthOptions, opts *Options) {
	if opts.AuthURL != "" {
		authOptions.IdentityEndpoint = opts.AuthURL
	}
	if opts.ProjectID != "" {
		authOptions.TenantID = opts.ProjectID
	}
	if opts.ProjectName != "" {
		authOptions.TenantName = opts.ProjectName
	}
	if opts.Username != "" {
		authOptions.Username = opts.Username
	}
	if opts.UserID != "" {
		authOptions.UserID = opts.UserID
	}
	if opts.Password != "" {
		authOptions.Password = opts.Password
	}
	if opts.Token != "" {
		authOptions.TokenID = opts.Token
	}
	if opts.ApplicationCredentialID != "" {
		authOptions.ApplicationCredentialID = opts.ApplicationCredentialID
	}
	if opts.ApplicationCredentialName != "" {
		authOptions.ApplicationCredentialName = opts.ApplicationCredentialName
	}
	if opts.ApplicationCredentialSecret != "" {
		authOptions.ApplicationCredentialSecret = opts.ApplicationCredentialSecret
	}
}

func endpointOptionsFromOptions(opts *Options) gophercloud.EndpointOpts {
	return gophercloud.EndpointOpts{
		Region:       firstNonEmpty(opts.RegionName, os.Getenv("OS_REGION_NAME")),
		Availability: availabilityFromInterface(firstNonEmpty(opts.Interface, os.Getenv("OS_INTERFACE"))),
	}
}

func availabilityFromInterface(endpointType string) gophercloud.Availability {
	switch endpointType {
	case "internal", "internalURL":
		return gophercloud.AvailabilityInternal
	case "admin", "adminURL":
		return gophercloud.AvailabilityAdmin
	default:
		return gophercloud.AvailabilityPublic
	}
}

func tokenIssueRowFromV3(result interface {
	ExtractToken() (*tokens3.Token, error)
	ExtractUser() (*tokens3.User, error)
	ExtractProject() (*tokens3.Project, error)
}) (tokenIssueRow, error) {
	token, err := result.ExtractToken()
	if err != nil {
		return tokenIssueRow{}, err
	}
	user, err := result.ExtractUser()
	if err != nil {
		return tokenIssueRow{}, err
	}
	project, err := result.ExtractProject()
	if err != nil {
		project = &tokens3.Project{}
	}

	return tokenIssueRow{
		Expires:   token.ExpiresAt.UTC().Format("2006-01-02T15:04:05-0700"),
		ID:        token.ID,
		ProjectID: project.ID,
		UserID:    user.ID,
	}, nil
}

func renderTokenIssue(stdout io.Writer, opts *Options, row tokenIssueRow) error {
	switch opts.Format {
	case "json":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(row)
	case "value":
		_, err := fmt.Fprintf(stdout, "%s\n%s\n%s\n%s\n", row.Expires, row.ID, row.ProjectID, row.UserID)
		return err
	case "pretty":
		_, err := fmt.Fprintf(stdout, "Token\n  Expires: %s\n  ID: %s\n  Project: %s\n  User: %s\n", row.Expires, row.ID, row.ProjectID, row.UserID)
		return err
	default:
		return renderFieldValueTable(stdout, opts, map[string]string{
			"expires":    row.Expires,
			"id":         row.ID,
			"project_id": row.ProjectID,
			"user_id":    row.UserID,
		})
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
