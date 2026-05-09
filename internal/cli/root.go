package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/crandallnet/golang-osc/compat/osc"
	_ "github.com/crandallnet/golang-osc/internal/plugins/cinderextras"
	_ "github.com/crandallnet/golang-osc/internal/plugins/local"
	_ "github.com/crandallnet/golang-osc/internal/plugins/neutronextras"
	_ "github.com/crandallnet/golang-osc/internal/plugins/novaextras"
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
	Format    string
	Pretty    bool
	Compact   bool
	NoCompact bool
	Debug     bool

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
	err := cmd.Execute()
	if err == nil {
		return nil
	}
	if handled, ok := handleParserError(err, args, stdout, stderr); ok {
		return handled
	}
	return err
}

type cliExitError struct {
	code   int
	silent bool
}

func (e *cliExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func (e *cliExitError) ExitCode() int {
	return e.code
}

func (e *cliExitError) Silent() bool {
	return e.silent
}

type parserOutputError struct {
	code    int
	stream  string
	message string
}

func (e *parserOutputError) Error() string {
	return strings.TrimRight(e.message, "\n")
}

func (e *parserOutputError) ExitCode() int {
	return e.code
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit interface {
		ExitCode() int
	}
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return 1
}

func ShouldPrintError(err error) bool {
	if err == nil {
		return false
	}
	var silent interface {
		Silent() bool
	}
	if errors.As(err, &silent) && silent.Silent() {
		return false
	}
	return true
}

func handleParserError(err error, args []string, stdout io.Writer, stderr io.Writer) (error, bool) {
	var parserErr *parserOutputError
	if errors.As(err, &parserErr) {
		if parserErr.stream == "stdout" {
			_, _ = fmt.Fprint(stdout, parserErr.message)
		} else {
			_, _ = fmt.Fprint(stderr, parserErr.message)
		}
		return &cliExitError{code: parserErr.code, silent: true}, true
	}
	if isCobraUnknownCommand(err) && len(args) > 0 {
		groups, loadErr := osc.Commands()
		if loadErr == nil {
			_, _ = fmt.Fprint(stdout, unknownCommandMessage(args, groups))
			return &cliExitError{code: 2, silent: true}, true
		}
	}
	return err, false
}

func NewRootCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	opts := &Options{
		Format:   defaultOutputFormat,
		Pretty:   envBoolInt("OS_PRETTY"),
		Compact:  envBoolInt("OS_COMPACT"),
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
	root.SetHelpFunc(func(command *cobra.Command, args []string) {
		if help, ok, err := osc.Help(""); err == nil && ok {
			fmt.Fprint(stdout, help)
		}
	})
	root.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return parserFlagError("", err)
	})
	root.Flags().SortFlags = false
	root.PersistentFlags().SortFlags = false
	addGlobalFlags(root.PersistentFlags(), opts)

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		prettyFlagChanged := cmd.Flags().Changed("pretty")
		formatFlagChanged := cmd.Flags().Changed("format")
		if opts.Pretty && (!formatFlagChanged || prettyFlagChanged) {
			opts.Format = "pretty"
		}
		if cmd.Flags().Changed("no-compact") {
			opts.Compact = false
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
	flags.BoolVar(&opts.Pretty, "pretty", opts.Pretty, "use enhanced human-readable output")
	flags.BoolVar(&opts.Compact, "compact", opts.Compact, "compact enhanced human-readable output")
	flags.BoolVar(&opts.NoCompact, "no-compact", false, "disable compact enhanced human-readable output")
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

func parserFlagError(path string, err error) error {
	message := formatFlagError(path, err)
	return &parserOutputError{code: 2, stream: "stderr", message: message}
}

func formatFlagError(path string, err error) string {
	usage := usageBlock(path)
	if usage != "" {
		usage += "\n"
	}
	flagText := unrecognizedArgument(err)
	commandName := "openstack"
	if strings.TrimSpace(path) != "" {
		commandName += " " + path
	}
	return usage + fmt.Sprintf("%s: error: unrecognized arguments: %s\n", commandName, flagText)
}

func usageBlock(path string) string {
	help, ok, err := osc.Help(path)
	if err != nil || !ok {
		return ""
	}
	help = strings.ReplaceAll(help, "\r\n", "\n")
	help = strings.ReplaceAll(help, "\r", "\n")
	if index := strings.Index(help, "\n\n"); index >= 0 {
		return strings.TrimRight(help[:index], "\n")
	}
	return strings.TrimRight(help, "\n")
}

func unrecognizedArgument(err error) string {
	message := err.Error()
	if strings.HasPrefix(message, "unknown flag: ") {
		return strings.TrimPrefix(message, "unknown flag: ")
	}
	if strings.HasPrefix(message, "unknown shorthand flag: ") {
		fields := strings.Fields(message)
		if len(fields) >= 5 {
			return "-" + strings.Trim(fields[3], "'\"")
		}
	}
	if strings.HasPrefix(message, "flag needs an argument: ") {
		return strings.TrimPrefix(message, "flag needs an argument: ")
	}
	return message
}

func isCobraUnknownCommand(err error) bool {
	message := err.Error()
	return strings.HasPrefix(message, "unknown command ")
}

func unknownCommandMessage(args []string, groups []osc.CommandGroup) string {
	commandText := strings.Join(args, " ")
	var builder strings.Builder
	fmt.Fprintf(&builder, "openstack: '%s' is not an openstack command. See 'openstack --help'.\n", commandText)
	matches := fuzzyCommandMatches(args[0], groups)
	if len(matches) > 0 {
		builder.WriteString("Did you mean one of these?\n")
		for _, match := range matches {
			fmt.Fprintf(&builder, "  %s\n", match)
		}
	}
	return builder.String()
}

func fuzzyCommandMatches(command string, groups []osc.CommandGroup) []string {
	commands := catalogCommandNames(groups)
	distances := make([]commandDistance, 0, len(commands))
	for _, candidate := range commands {
		prefix := strings.Fields(candidate)[0]
		if strings.HasPrefix(candidate, command) {
			distances = append(distances, commandDistance{Distance: 0, Command: candidate})
			continue
		}
		distances = append(distances, commandDistance{
			Distance: damerauLevenshtein(command, prefix) + 1,
			Command:  candidate,
		})
	}
	sort.Slice(distances, func(i int, j int) bool {
		if distances[i].Distance == distances[j].Distance {
			return distances[i].Command < distances[j].Command
		}
		return distances[i].Distance < distances[j].Distance
	})
	var matches []string
	matchDistance := 0
	for _, distance := range distances {
		if distance.Distance > matchDistance {
			if matchDistance != 0 {
				break
			}
			matchDistance = distance.Distance
		}
		matches = append(matches, distance.Command)
	}
	return matches
}

type commandDistance struct {
	Distance int
	Command  string
}

func catalogCommandNames(groups []osc.CommandGroup) []string {
	seen := map[string]bool{}
	var commands []string
	for _, group := range groups {
		for _, command := range group.Commands {
			if seen[command] {
				continue
			}
			seen[command] = true
			commands = append(commands, command)
		}
	}
	sort.Strings(commands)
	return commands
}

func damerauLevenshtein(a string, b string) int {
	const (
		swapCost         = 0
		substitutionCost = 2
		insertionCost    = 1
		deletionCost     = 3
	)
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b) * insertionCost
	}
	if len(b) == 0 {
		return len(a) * deletionCost
	}

	row1 := make([]int, len(b)+1)
	row2 := make([]int, len(b)+1)
	row0 := make([]int, len(b)+1)
	for i := range row1 {
		row1[i] = i * insertionCost
		row2[i] = row1[i]
		row0[i] = row1[i]
	}

	for i := range a {
		row2[0] = (i + 1) * deletionCost
		for j := range b {
			substitution := row1[j]
			if a[i] != b[j] {
				substitution += substitutionCost
			}
			insertion := row2[j] + insertionCost
			deletion := row1[j+1] + deletionCost
			cost := minInt(substitution, insertion, deletion)
			if i > 0 && j > 0 && a[i-1] == b[j] && a[i] == b[j-1] {
				cost = minInt(cost, row0[j-1]+swapCost)
			}
			row2[j+1] = cost
		}
		row0, row1, row2 = row1, row2, row0
	}
	return row1[len(b)]
}

func minInt(first int, values ...int) int {
	minimum := first
	for _, value := range values {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
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
