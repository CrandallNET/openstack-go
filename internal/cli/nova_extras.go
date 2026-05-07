package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

type serverSSHRequest struct {
	ServerName            string
	Address               string
	Port                  int
	User                  string
	IdentityFiles         []string
	KnownHostsFiles       []string
	StrictHostKeyChecking string
	ForwardAgent          bool
	DisableAgent          bool
	ForceTTY              bool
	DisableTTY            bool
	BatchMode             bool
	Verbose               bool
	RemoteCommand         []string
	Stdin                 io.Reader
	Stdout                io.Writer
	Stderr                io.Writer
}

type serverSSHSessionRunner func(context.Context, serverSSHRequest) error

var runServerSSHSession serverSSHSessionRunner = runPureGoSSHSession

func isNovaExtrasCommand(path string) bool {
	return path == "server ssh"
}

func serverSSHPassThroughHelp() string {
	return `
Pure Go SSH pass-through options:
  Pass these after --. Unsupported OpenSSH options return an error instead of
  being silently ignored.

  -l <login-name>                         SSH login name
  -p <port>                               SSH port
  -i <keyfile>                            SSH private key file
  -A                                      enable ssh-agent forwarding
  -a                                      disable ssh-agent use and forwarding
  -t                                      force pseudo-terminal allocation
  -T                                      disable pseudo-terminal allocation
  -v, -vv, -vvv                           enable verbose SSH mode
  -o User=<login-name>                    SSH login name
  -o Port=<port>                          SSH port
  -o IdentityFile=<keyfile>               SSH private key file
  -o StrictHostKeyChecking=yes|no|ask|accept-new|off
  -o UserKnownHostsFile=<path> [<path>...]
  -o ForwardAgent=yes|no                  enable or disable agent forwarding
  -o BatchMode=yes|no                     disable interactive host-key prompt when yes
  -o IdentitiesOnly=yes|no                restrict authentication to identity files when yes
  -o PasswordAuthentication=yes|no        accepted for compatibility; password auth is not implemented
  -o KbdInteractiveAuthentication=yes|no  accepted for compatibility; keyboard-interactive auth is not implemented
  -o LogLevel=<level>                     accepted for compatibility
  <remote-command> [args ...]             execute a remote command instead of an interactive shell

Pure Go SSH defaults:
  OS_SSH_USER=<login-name>                default SSH login name; overridden by --login, -l, or -o User
`
}

func runNovaExtras(path string, stdout io.Writer, opts *Options) commandHandler {
	return func(cmd *cobra.Command, args []string) error {
		clients, err := newOpenStackClients(cmd.Context(), opts)
		if err != nil {
			return err
		}
		client, err := clients.computeV2()
		if err != nil {
			return err
		}
		switch path {
		case "server ssh":
			return serverSSH(cmd.Context(), stdout, opts, clients.AuthOptions, client, args)
		default:
			return fmt.Errorf("unsupported nova extras command %q", path)
		}
	}
}

func serverSSH(ctx context.Context, stdout io.Writer, opts *Options, authOptions gophercloud.AuthOptions, client *gophercloud.ServiceClient, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("server ssh requires <server>")
	}
	server, err := findServer(ctx, client, args[0])
	if err != nil {
		return err
	}
	request, err := buildServerSSHRequest(stdout, opts, authOptions, server, args)
	if err != nil {
		return err
	}
	return runServerSSHSession(ctx, request)
}

func buildServerSSHRequest(stdout io.Writer, opts *Options, authOptions gophercloud.AuthOptions, server *servers.Server, args []string) (serverSSHRequest, error) {
	invocation, err := parseServerSSHInvocation(opts, authOptions, args)
	if err != nil {
		return serverSSHRequest{}, err
	}
	address, err := serverSSHAddressForInvocation(server.Addresses, invocation)
	if err != nil {
		return serverSSHRequest{}, err
	}
	return serverSSHRequest{
		ServerName:            server.Name,
		Address:               address,
		Port:                  invocation.port,
		User:                  invocation.login,
		IdentityFiles:         invocation.identityFiles,
		KnownHostsFiles:       invocation.knownHostsFiles,
		StrictHostKeyChecking: invocation.strictHostKeyChecking,
		ForwardAgent:          invocation.forwardAgent,
		DisableAgent:          invocation.disableAgent,
		ForceTTY:              invocation.forceTTY,
		DisableTTY:            invocation.disableTTY,
		BatchMode:             invocation.batchMode,
		Verbose:               invocation.verbose,
		RemoteCommand:         invocation.remoteCommand,
		Stdin:                 os.Stdin,
		Stdout:                stdout,
		Stderr:                os.Stderr,
	}, nil
}

type serverSSHInvocation struct {
	login                 string
	port                  int
	identityFiles         []string
	knownHostsFiles       []string
	strictHostKeyChecking string
	addressType           string
	addressTypeExplicit   bool
	ipFamilies            []int
	forwardAgent          bool
	disableAgent          bool
	forceTTY              bool
	disableTTY            bool
	batchMode             bool
	verbose               bool
	remoteCommand         []string
}

func parseServerSSHInvocation(opts *Options, authOptions gophercloud.AuthOptions, args []string) (serverSSHInvocation, error) {
	if len(args) < 1 {
		return serverSSHInvocation{}, fmt.Errorf("server ssh requires <server>")
	}
	invocation := serverSSHInvocation{
		login:                 firstNonEmpty(flagValue(opts, "login"), os.Getenv("OS_SSH_USER"), authOptions.Username, os.Getenv("USER"), os.Getenv("USERNAME")),
		port:                  22,
		addressType:           "public",
		ipFamilies:            []int{4, 6},
		strictHostKeyChecking: "ask",
	}
	if port := intFlag(opts, "port"); port > 0 {
		invocation.port = port
	}
	if identity := flagValue(opts, "identity"); identity != "" {
		invocation.identityFiles = append(invocation.identityFiles, identity)
	}
	if option := flagValue(opts, "option"); option != "" {
		if err := applyServerSSHOption(&invocation, option); err != nil {
			return serverSSHInvocation{}, err
		}
	}
	if boolFlag(opts, "verbose") {
		invocation.verbose = true
	}
	if boolFlag(opts, "ipv4") && boolFlag(opts, "ipv6") {
		return serverSSHInvocation{}, fmt.Errorf("argument -6: not allowed with argument -4")
	}
	if boolFlag(opts, "ipv4") {
		invocation.ipFamilies = []int{4}
	}
	if boolFlag(opts, "ipv6") {
		invocation.ipFamilies = []int{6}
	}
	addressTypeFlags := 0
	if boolFlag(opts, "public") {
		addressTypeFlags++
		invocation.addressType = "public"
		invocation.addressTypeExplicit = true
	}
	if boolFlag(opts, "private") {
		addressTypeFlags++
		invocation.addressType = "private"
		invocation.addressTypeExplicit = true
	}
	if addressType := flagValue(opts, "address-type"); addressType != "" {
		addressTypeFlags++
		invocation.addressType = addressType
		invocation.addressTypeExplicit = true
	}
	if addressTypeFlags > 1 {
		return serverSSHInvocation{}, fmt.Errorf("argument --address-type: not allowed with argument --public or --private")
	}
	if err := parseServerSSHPassThrough(&invocation, args[1:]); err != nil {
		return serverSSHInvocation{}, err
	}
	if invocation.port <= 0 || invocation.port > 65535 {
		return serverSSHInvocation{}, fmt.Errorf("invalid SSH port %d", invocation.port)
	}
	return invocation, nil
}

func parseServerSSHPassThrough(invocation *serverSSHInvocation, args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			continue
		}
		if arg == "" {
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			invocation.remoteCommand = append(invocation.remoteCommand, args[i:]...)
			return nil
		}
		switch {
		case arg == "-l":
			value, next, err := serverSSHNextArg(args, i, "-l")
			if err != nil {
				return err
			}
			invocation.login = value
			i = next
		case strings.HasPrefix(arg, "-l") && len(arg) > 2:
			invocation.login = arg[2:]
		case arg == "-p":
			value, next, err := serverSSHNextArg(args, i, "-p")
			if err != nil {
				return err
			}
			port, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("argument -p: invalid int value: %q", value)
			}
			invocation.port = port
			i = next
		case strings.HasPrefix(arg, "-p") && len(arg) > 2:
			port, err := strconv.Atoi(arg[2:])
			if err != nil {
				return fmt.Errorf("argument -p: invalid int value: %q", arg[2:])
			}
			invocation.port = port
		case arg == "-i":
			value, next, err := serverSSHNextArg(args, i, "-i")
			if err != nil {
				return err
			}
			invocation.identityFiles = append(invocation.identityFiles, value)
			i = next
		case strings.HasPrefix(arg, "-i") && len(arg) > 2:
			invocation.identityFiles = append(invocation.identityFiles, arg[2:])
		case arg == "-o":
			value, next, err := serverSSHNextArg(args, i, "-o")
			if err != nil {
				return err
			}
			if err := applyServerSSHOption(invocation, value); err != nil {
				return err
			}
			i = next
		case strings.HasPrefix(arg, "-o") && len(arg) > 2:
			if err := applyServerSSHOption(invocation, arg[2:]); err != nil {
				return err
			}
		case arg == "-A":
			invocation.forwardAgent = true
			invocation.disableAgent = false
		case arg == "-a":
			invocation.disableAgent = true
			invocation.forwardAgent = false
		case arg == "-t":
			invocation.forceTTY = true
			invocation.disableTTY = false
		case arg == "-T":
			invocation.disableTTY = true
			invocation.forceTTY = false
		case strings.HasPrefix(arg, "-v"):
			invocation.verbose = true
		default:
			return fmt.Errorf("unsupported pure Go SSH option %q; supported pass-through options are -l, -p, -i, -o, -A, -a, -t, -T, -v, and a remote command", arg)
		}
	}
	return nil
}

func serverSSHNextArg(args []string, index int, flag string) (string, int, error) {
	next := index + 1
	if next >= len(args) {
		return "", index, fmt.Errorf("argument %s: expected one argument", flag)
	}
	return args[next], next, nil
}

func applyServerSSHOption(invocation *serverSSHInvocation, value string) error {
	key, optionValue, ok := strings.Cut(value, "=")
	if !ok {
		fields := strings.Fields(value)
		if len(fields) != 2 {
			return fmt.Errorf("argument -o: expected key=value, got %q", value)
		}
		key, optionValue = fields[0], fields[1]
	}
	key = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", ""))
	optionValue = strings.Trim(strings.TrimSpace(optionValue), `"`)
	switch key {
	case "user":
		invocation.login = optionValue
	case "port":
		port, err := strconv.Atoi(optionValue)
		if err != nil {
			return fmt.Errorf("argument -o Port: invalid int value: %q", optionValue)
		}
		invocation.port = port
	case "identityfile":
		invocation.identityFiles = append(invocation.identityFiles, optionValue)
	case "stricthostkeychecking":
		normalized := strings.ToLower(optionValue)
		switch normalized {
		case "yes", "no", "ask", "accept-new", "off":
			if normalized == "off" {
				normalized = "no"
			}
			invocation.strictHostKeyChecking = normalized
		default:
			return fmt.Errorf("argument -o StrictHostKeyChecking: invalid value %q", optionValue)
		}
	case "userknownhostsfile":
		invocation.knownHostsFiles = strings.Fields(optionValue)
	case "forwardagent":
		invocation.forwardAgent = sshOptionTruthy(optionValue)
		invocation.disableAgent = !invocation.forwardAgent
	case "batchmode":
		invocation.batchMode = sshOptionTruthy(optionValue)
	case "identitiesonly":
		if sshOptionTruthy(optionValue) {
			invocation.disableAgent = true
		}
	case "passwordauthentication", "kbdinteractiveauthentication", "loglevel":
		return nil
	default:
		return fmt.Errorf("unsupported pure Go SSH option -o %s; supported options are User, Port, IdentityFile, StrictHostKeyChecking, UserKnownHostsFile, ForwardAgent, BatchMode, IdentitiesOnly, PasswordAuthentication, KbdInteractiveAuthentication, and LogLevel", key)
	}
	return nil
}

func sshOptionTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "true", "on", "1":
		return true
	default:
		return false
	}
}

func serverSSHAddressForInvocation(addresses map[string]any, invocation serverSSHInvocation) (string, error) {
	address, err := serverSSHAddress(addresses, invocation.addressType, invocation.ipFamilies)
	if err == nil {
		return address, nil
	}
	if invocation.addressTypeExplicit || invocation.addressType != "public" {
		return "", err
	}
	if privateAddress, privateErr := serverSSHAddress(addresses, "private", invocation.ipFamilies); privateErr == nil {
		return privateAddress, nil
	}
	return "", err
}

func serverSSHAddress(addresses map[string]any, addressType string, ipFamilies []int) (string, error) {
	families := map[int]bool{}
	for _, family := range ipFamilies {
		families[family] = true
	}
	if rawAddresses, ok := addresses[addressType]; ok {
		for _, address := range serverSSHAddressEntries(rawAddresses) {
			if families[serverSSHAddressVersion(address)] {
				if value := serverSSHAddressValue(address); value != "" {
					return value, nil
				}
			}
		}
	}
	newAddressType := addressType
	if addressType == "public" {
		newAddressType = "floating"
	}
	if addressType == "private" {
		newAddressType = "fixed"
	}
	for _, network := range sortedKeysFromAnyMap(addresses) {
		entries := serverSSHAddressEntries(addresses[network])
		for _, address := range entries {
			if _, ok := address.(string); ok {
				if len(entries) == 0 {
					continue
				}
				if newAddressType == "fixed" {
					return serverSSHStringAddress(entries[0]), nil
				}
				return serverSSHStringAddress(entries[len(entries)-1]), nil
			}
			entryType := serverSSHAddressType(address)
			if entryType != newAddressType {
				continue
			}
			if families[serverSSHAddressVersion(address)] {
				if value := serverSSHAddressValue(address); value != "" {
					return value, nil
				}
			}
		}
	}
	return "", fmt.Errorf("ERROR: No %s IP version %v address found", addressType, ipFamilies)
}

func serverSSHAddressEntries(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		entries := make([]any, 0, len(typed))
		for _, entry := range typed {
			entries = append(entries, entry)
		}
		return entries
	case []string:
		entries := make([]any, 0, len(typed))
		for _, entry := range typed {
			entries = append(entries, entry)
		}
		return entries
	default:
		return nil
	}
}

func serverSSHAddressValue(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if addr, ok := typed["addr"].(string); ok {
			return addr
		}
	case map[string]string:
		return typed["addr"]
	}
	return ""
}

func serverSSHStringAddress(value any) string {
	if stringAddress, ok := value.(string); ok {
		return stringAddress
	}
	return serverSSHAddressValue(value)
}

func serverSSHAddressType(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if addressType, ok := typed["OS-EXT-IPS:type"].(string); ok {
			return addressType
		}
	case map[string]string:
		return typed["OS-EXT-IPS:type"]
	}
	return ""
}

func serverSSHAddressVersion(value any) int {
	switch typed := value.(type) {
	case map[string]any:
		parsed, _ := intFromAny(typed["version"])
		return parsed
	case map[string]string:
		parsed, _ := strconv.Atoi(typed["version"])
		return parsed
	}
	return 0
}

func sortedKeysFromAnyMap(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func runPureGoSSHSession(ctx context.Context, request serverSSHRequest) error {
	if request.Port == 0 {
		request.Port = 22
	}
	if request.Stdin == nil {
		request.Stdin = os.Stdin
	}
	if request.Stdout == nil {
		request.Stdout = os.Stdout
	}
	if request.Stderr == nil {
		request.Stderr = os.Stderr
	}
	if request.User == "" {
		return fmt.Errorf("SSH username is required; use -l/--login or -o User=<name>")
	}
	authMethods, agentClient, closeAgent, err := serverSSHAuthMethods(request)
	if err != nil {
		return err
	}
	if closeAgent != nil {
		defer closeAgent()
	}
	if len(authMethods) == 0 {
		return fmt.Errorf("no SSH authentication methods available; use -i/--identity, ssh-agent, or a supported key in ~/.ssh")
	}
	hostKeyCallback, err := serverSSHHostKeyCallback(request)
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User:            request.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}
	address := net.JoinHostPort(request.Address, strconv.Itoa(request.Port))
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		return err
	}
	client := ssh.NewClient(clientConn, chans, reqs)
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin = request.Stdin
	session.Stdout = request.Stdout
	session.Stderr = request.Stderr
	if request.ForwardAgent && agentClient != nil {
		if err := agent.RequestAgentForwarding(session); err != nil {
			return err
		}
		if err := agent.ForwardToAgent(client, agentClient); err != nil {
			return err
		}
	}
	if len(request.RemoteCommand) > 0 {
		return serverSSHRunError(session.Run(strings.Join(request.RemoteCommand, " ")))
	}
	restore, err := requestSSHPTY(session, request)
	if err != nil {
		return err
	}
	if restore != nil {
		defer restore()
	}
	if err := session.Shell(); err != nil {
		return err
	}
	return serverSSHRunError(session.Wait())
}

func serverSSHAuthMethods(request serverSSHRequest) ([]ssh.AuthMethod, agent.Agent, func(), error) {
	var methods []ssh.AuthMethod
	var agentClient agent.Agent
	var closeAgent func()
	identityFiles := request.IdentityFiles
	specifiedIdentity := len(identityFiles) > 0
	if len(identityFiles) == 0 {
		identityFiles = defaultSSHIdentityFiles()
	}
	for _, identityFile := range identityFiles {
		path := expandUserPath(identityFile)
		key, err := os.ReadFile(path)
		if err != nil {
			if specifiedIdentity {
				return nil, nil, nil, err
			}
			continue
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			if specifiedIdentity {
				return nil, nil, nil, fmt.Errorf("parse SSH identity %s: %w", path, err)
			}
			continue
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if !request.DisableAgent {
		if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
			conn, err := net.Dial("unix", sock)
			if err == nil {
				agentClient = agent.NewClient(conn)
				methods = append(methods, ssh.PublicKeysCallback(agentClient.Signers))
				closeAgent = func() { _ = conn.Close() }
			}
		}
	}
	return methods, agentClient, closeAgent, nil
}

func defaultSSHIdentityFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_ecdsa_sk"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
	var files []string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			files = append(files, candidate)
		}
	}
	return files
}

func serverSSHHostKeyCallback(request serverSSHRequest) (ssh.HostKeyCallback, error) {
	strict := request.StrictHostKeyChecking
	if strict == "" {
		strict = "ask"
	}
	if strict == "no" {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	files := request.KnownHostsFiles
	if len(files) == 0 {
		files = defaultKnownHostsFiles()
	}
	var existing []string
	for _, file := range files {
		path := expandUserPath(file)
		if path == "" || path == "none" || path == "/dev/null" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	var knownCallback ssh.HostKeyCallback
	if len(existing) > 0 {
		callback, err := knownhosts.New(existing...)
		if err != nil {
			return nil, err
		}
		knownCallback = callback
	}
	appendFile := ""
	if len(files) > 0 {
		appendFile = expandUserPath(files[0])
	} else if defaults := defaultKnownHostsFiles(); len(defaults) > 0 {
		appendFile = defaults[0]
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if knownCallback != nil {
			err := knownCallback(hostname, remote, key)
			if err == nil {
				return nil
			}
			var keyErr *knownhosts.KeyError
			if !errors.As(err, &keyErr) || len(keyErr.Want) > 0 {
				return err
			}
		}
		switch strict {
		case "yes":
			return fmt.Errorf("host key verification failed for %s", hostname)
		case "accept-new":
			return appendKnownHost(appendFile, hostname, key)
		case "ask":
			if request.BatchMode {
				return fmt.Errorf("host key verification failed for %s", hostname)
			}
			accepted, err := promptAcceptHostKey(request, hostname, key)
			if err != nil {
				return err
			}
			if !accepted {
				return fmt.Errorf("host key verification failed for %s", hostname)
			}
			return appendKnownHost(appendFile, hostname, key)
		default:
			return fmt.Errorf("unsupported StrictHostKeyChecking value %q", strict)
		}
	}, nil
}

func defaultKnownHostsFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".ssh", "known_hosts"), filepath.Join(home, ".ssh", "known_hosts2")}
}

func promptAcceptHostKey(request serverSSHRequest, hostname string, key ssh.PublicKey) (bool, error) {
	stdinFile, ok := request.Stdin.(*os.File)
	if !ok || !term.IsTerminal(int(stdinFile.Fd())) {
		return false, fmt.Errorf("host %s is not in known_hosts and stdin is not interactive", hostname)
	}
	_, _ = fmt.Fprintf(request.Stderr, "The authenticity of host %q can't be established.\n%s key fingerprint is %s.\nAre you sure you want to continue connecting (yes/no)? ", hostname, key.Type(), ssh.FingerprintSHA256(key))
	reader := bufio.NewReader(request.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "yes" || answer == "y", nil
}

func appendKnownHost(path string, hostname string, key ssh.PublicKey) error {
	if path == "" || path == "none" || path == "/dev/null" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key) + "\n")
	return err
}

func requestSSHPTY(session *ssh.Session, request serverSSHRequest) (func(), error) {
	if request.DisableTTY {
		return nil, nil
	}
	stdinFile, stdinOK := request.Stdin.(*os.File)
	stdoutFile, stdoutOK := request.Stdout.(*os.File)
	if !request.ForceTTY && (!stdinOK || !stdoutOK || !term.IsTerminal(int(stdinFile.Fd())) || !term.IsTerminal(int(stdoutFile.Fd()))) {
		return nil, nil
	}
	width, height := 80, 24
	if stdoutOK {
		if detectedWidth, detectedHeight, err := term.GetSize(int(stdoutFile.Fd())); err == nil {
			width, height = detectedWidth, detectedHeight
		}
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	terminalName := firstNonEmpty(os.Getenv("TERM"), "xterm")
	if err := session.RequestPty(terminalName, height, width, modes); err != nil {
		return nil, err
	}
	if stdinOK && term.IsTerminal(int(stdinFile.Fd())) {
		oldState, err := term.MakeRaw(int(stdinFile.Fd()))
		if err != nil {
			return nil, err
		}
		return func() {
			_ = term.Restore(int(stdinFile.Fd()), oldState)
		}, nil
	}
	return nil, nil
}

func serverSSHRunError(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		return &cliExitError{code: exitErr.ExitStatus(), silent: true}
	}
	return err
}
