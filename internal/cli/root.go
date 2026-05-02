package cli

import (
	"fmt"
	"io"

	"github.com/crandallnet/golang-osc/compat/osc"
	_ "github.com/crandallnet/golang-osc/internal/plugins/local"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	CLIVersion                 = "0.0.0-dev"
	OSCCompatibilityTarget     = "9.0.0"
	defaultOutputFormat        = "table"
	notImplementedExitCodeText = "This command is not yet implemented"
)

type Options struct {
	Format string
	Pretty bool
	Debug  bool
}

func Execute(args []string, stdout io.Writer, stderr io.Writer) error {
	cmd := NewRootCommand(stdout, stderr)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func NewRootCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	opts := &Options{Format: defaultOutputFormat}
	groups, err := osc.Commands()
	if err != nil {
		panic(fmt.Sprintf("load embedded OSC command catalog: %v", err))
	}

	root := &cobra.Command{
		Use:           "openstack",
		Short:         "OpenStack command line client",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       CLIVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetVersionTemplate("openstack {{.Version}}\n")
	root.Flags().SortFlags = false
	root.PersistentFlags().SortFlags = false
	addGlobalFlags(root.PersistentFlags(), opts)

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if opts.Pretty {
			opts.Format = "pretty"
		}
		return nil
	}

	root.AddCommand(newHelpCommand(root, stdout))
	root.AddCommand(newCompleteCommand(stdout))
	newCommandRegistry(groups, stdout, opts).addCatalogCommands(root)

	return root
}

func addGlobalFlags(flags *pflag.FlagSet, opts *Options) {
	flags.StringVarP(&opts.Format, "format", "f", defaultOutputFormat, "the output format")
	flags.BoolVar(&opts.Pretty, "pretty", false, "use enhanced human-readable output")
	flags.BoolVar(&opts.Debug, "debug", false, "show tracebacks on errors")

	flags.String("os-cloud", "", "cloud name in clouds.yaml")
	flags.String("os-auth-url", "", "authentication URL")
	flags.String("os-project-name", "", "project name")
	flags.String("os-project-id", "", "project ID")
	flags.String("os-project-domain-name", "", "project domain name")
	flags.String("os-project-domain-id", "", "project domain ID")
	flags.String("os-username", "", "username")
	flags.String("os-user-id", "", "user ID")
	flags.String("os-user-domain-name", "", "user domain name")
	flags.String("os-user-domain-id", "", "user domain ID")
	flags.String("os-password", "", "password")
	flags.String("os-token", "", "token")
	flags.String("os-region-name", "", "region name")
	flags.String("os-interface", "", "interface type")
	flags.Bool("os-insecure", false, "disable TLS certificate verification")
}

func newHelpCommand(root *cobra.Command, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Show help for a command",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, _, err := root.Find(args)
			if err != nil {
				return err
			}
			if target == nil {
				target = root
			}
			target.SetOut(stdout)
			return target.Help()
		},
	}
}

func newCompleteCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "complete",
		Short: "Print shell completion functions",
		RunE: func(cmd *cobra.Command, args []string) error {
			completion, err := osc.Completion()
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(stdout, completion)
			return err
		},
	}
}
