package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"

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

	MaxWidth   int
	FitWidth   bool
	PrintEmpty bool
	NoIndent   bool
	Columns    []string
	Prefix     string
	Quote      string

	SortColumns     []string
	SortAscending   bool
	SortDescending  bool
	CommandFlags    map[string]string
	CommandFlagList map[string][]string

	Cloud                       string
	AuthURL                     string
	ProjectName                 string
	ProjectID                   string
	ProjectDomainName           string
	ProjectDomainID             string
	Username                    string
	UserID                      string
	UserDomainName              string
	UserDomainID                string
	Password                    string
	Token                       string
	RegionName                  string
	Interface                   string
	Insecure                    bool
	ApplicationCredentialID     string
	ApplicationCredentialName   string
	ApplicationCredentialSecret string
}

func Execute(args []string, stdout io.Writer, stderr io.Writer) error {
	cmd := NewRootCommand(stdout, stderr)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func NewRootCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	opts := &Options{
		Format:   defaultOutputFormat,
		MaxWidth: envInt("CLIFF_MAX_TERM_WIDTH"),
		FitWidth: envBoolInt("CLIFF_FIT_WIDTH"),
	}
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
		opts.CommandFlags = commandFlagValues(cmd)
		opts.CommandFlagList = commandFlagLists(cmd)
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
	flags.IntVar(&opts.MaxWidth, "max-width", opts.MaxWidth, "maximum display width, <1 to disable")
	flags.BoolVar(&opts.FitWidth, "fit-width", opts.FitWidth, "fit table output to the display width")
	flags.BoolVar(&opts.PrintEmpty, "print-empty", false, "print empty table if there is no data to show")
	flags.BoolVar(&opts.NoIndent, "noindent", false, "disable indenting JSON and YAML output")
	flags.StringArrayVarP(&opts.Columns, "column", "c", nil, "specify the column(s) to include")
	flags.StringVar(&opts.Prefix, "prefix", "", "add a prefix to shell variable names")
	flags.StringVar(&opts.Quote, "quote", "nonnumeric", "when to include quotes in CSV output")
	flags.StringArrayVar(&opts.SortColumns, "sort-column", nil, "specify the column(s) to sort the data")
	flags.BoolVar(&opts.SortAscending, "sort-ascending", false, "sort the column(s) in ascending order")
	flags.BoolVar(&opts.SortDescending, "sort-descending", false, "sort the column(s) in descending order")

	flags.StringVar(&opts.Cloud, "os-cloud", "", "cloud name in clouds.yaml")
	flags.StringVar(&opts.AuthURL, "os-auth-url", "", "authentication URL")
	flags.StringVar(&opts.ProjectName, "os-project-name", "", "project name")
	flags.StringVar(&opts.ProjectID, "os-project-id", "", "project ID")
	flags.StringVar(&opts.ProjectDomainName, "os-project-domain-name", "", "project domain name")
	flags.StringVar(&opts.ProjectDomainID, "os-project-domain-id", "", "project domain ID")
	flags.StringVar(&opts.Username, "os-username", "", "username")
	flags.StringVar(&opts.UserID, "os-user-id", "", "user ID")
	flags.StringVar(&opts.UserDomainName, "os-user-domain-name", "", "user domain name")
	flags.StringVar(&opts.UserDomainID, "os-user-domain-id", "", "user domain ID")
	flags.StringVar(&opts.Password, "os-password", "", "password")
	flags.StringVar(&opts.Token, "os-token", "", "token")
	flags.StringVar(&opts.RegionName, "os-region-name", "", "region name")
	flags.StringVar(&opts.Interface, "os-interface", "", "interface type")
	flags.BoolVar(&opts.Insecure, "os-insecure", false, "disable TLS certificate verification")
	flags.StringVar(&opts.ApplicationCredentialID, "os-application-credential-id", "", "application credential ID")
	flags.StringVar(&opts.ApplicationCredentialName, "os-application-credential-name", "", "application credential name")
	flags.StringVar(&opts.ApplicationCredentialSecret, "os-application-credential-secret", "", "application credential secret")
}

func commandFlagValues(cmd *cobra.Command) map[string]string {
	values := map[string]string{}
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			values[flag.Name] = flag.Value.String()
		}
	})
	return values
}

func commandFlagLists(cmd *cobra.Command) map[string][]string {
	values := map[string][]string{}
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed {
			return
		}
		if sliceValue, ok := flag.Value.(pflag.SliceValue); ok {
			values[flag.Name] = sliceValue.GetSlice()
		}
	})
	return values
}

func envInt(name string) int {
	value := os.Getenv(name)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func envBoolInt(name string) bool {
	return envInt(name) != 0
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
