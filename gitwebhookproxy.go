package main

import (
	"crypto/fips140"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/schwichtgit/GitWebhookProxy/pkg/proxy"
	"github.com/spf13/pflag"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

// errUsage is returned when the command line is invalid and the usage text
// has already been printed.
var errUsage = errors.New("see flag usage")

type config struct {
	listenAddress      string
	upstreamURL        string
	secret             string
	provider           string
	allowedPaths       []string
	ignoredUsers       []string
	allowedUsers       []string
	insecureSkipVerify bool
}

// envFlagNames maps upper-cased GWP_* suffixes to flag names.
var envFlagNames = map[string]string{
	"UPSTREAMURL":        "upstreamURL",
	"ALLOWEDPATHS":       "allowedPaths",
	"IGNOREDUSERS":       "ignoredUsers",
	"ALLOWEDUSER":        "allowedUser",
	"PROVIDER":           "provider",
	"SECRET":             "secret",
	"LISTEN":             "listen",
	"INSECURESKIPVERIFY": "insecureSkipVerify",
}

// parseConfig reads flags from args (without the program name) and then
// fills any flag not set on the command line from GWP_* entries in environ.
func parseConfig(name string, args, environ []string, stdout, stderr io.Writer, logger *log.Logger) (*config, error) {
	fs := pflag.NewFlagSet(name, pflag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.ParseErrorsAllowlist.UnknownFlagsHandling = pflag.IgnoreUnknownFlag

	listenAddress := fs.String("listen", ":8080", "Address on which the proxy listens.")
	upstreamURL := fs.String("upstreamURL", "", "URL to which the proxy requests will be forwarded (required)")
	secret := fs.String("secret", "", "Secret of the Webhook API. If not set validation is not made.")
	provider := fs.String("provider", "github", "Git Provider which generates the Webhook")
	allowedPaths := fs.String("allowedPaths", "", "Comma-Separated String List of allowed paths")
	ignoredUsers := fs.String("ignoredUsers", "", "Comma-Separated String List of users to ignore while proxying Webhook request")
	allowedUsers := fs.String("allowedUser", "", "Comma-Separated String List of users to allow while proxying Webhook request")
	insecureTLS := fs.Bool("insecureSkipVerify", false, "Skip TLS certificate verification of the upstream. INSECURE: only enable for trusted upstreams using self-signed certificates.")

	usage := func() {
		_, _ = fmt.Fprintf(stderr, "Usage of %s:\n", name)
		fs.PrintDefaults()
	}

	// Mirror pflag.ExitOnError output: usage, then the error.
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return nil, err
		}
		usage()
		_, _ = fmt.Fprintln(stderr, err)
		return nil, errUsage
	}

	for _, env := range environ {
		if !strings.HasPrefix(env, "GWP_") {
			continue
		}
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envName := strings.TrimPrefix(parts[0], "GWP_")
		flagName, ok := envFlagNames[strings.ToUpper(envName)]
		if !ok {
			flagName = strings.ToLower(envName)
		}

		// Command line flags take precedence over environment variables.
		if flag := fs.Lookup(flagName); flag != nil && !flag.Changed {
			if err := flag.Value.Set(parts[1]); err != nil {
				logger.Printf("Error setting flag %s: %v", flagName, err)
			}
		}
	}

	if len(strings.TrimSpace(*upstreamURL)) == 0 {
		logger.Println("Required flag 'upstreamURL' not specified")
		_, _ = fmt.Fprintln(stdout, "")
		usage()
		_, _ = fmt.Fprintln(stdout, "")
		return nil, errUsage
	}

	return &config{
		listenAddress:      *listenAddress,
		upstreamURL:        *upstreamURL,
		secret:             *secret,
		provider:           strings.ToLower(*provider),
		allowedPaths:       splitList(*allowedPaths),
		ignoredUsers:       splitList(*ignoredUsers),
		allowedUsers:       splitList(*allowedUsers),
		insecureSkipVerify: *insecureTLS,
	}, nil
}

// splitList splits a comma-separated list. An empty string yields an empty,
// non-nil slice.
func splitList(s string) []string {
	if len(s) == 0 {
		return []string{}
	}
	return strings.Split(s, ",")
}

// run configures the proxy from args and environ, logs the startup state and
// hands the proxy to serve.
func run(args, environ []string, stdout, stderr io.Writer, serve func(p *proxy.Proxy, listenAddress string) error) error {
	logger := log.New(stderr, "", log.LstdFlags)

	name := "gitwebhookproxy"
	if len(args) > 0 {
		name, args = args[0], args[1:]
	}

	cfg, err := parseConfig(name, args, environ, stdout, stderr, logger)
	if err != nil {
		return err
	}

	logger.Printf("Git WebHook Proxy %s started with provider '%s'\n", version, cfg.provider)

	if fips140.Enabled() {
		logger.Printf("FIPS 140-3 mode enabled (Go Cryptographic Module %s)", fips140.Version())
	} else {
		logger.Println("FIPS 140-3 mode disabled (build with GOFIPS140=v1.0.0 or run with GODEBUG=fips140=on to enable)")
	}

	if cfg.insecureSkipVerify {
		logger.Println("WARNING: upstream TLS certificate verification is disabled (insecureSkipVerify=true)")
	}
	proxy.SetInsecureSkipVerify(cfg.insecureSkipVerify)

	p, err := proxy.NewProxy(cfg.upstreamURL, cfg.allowedPaths, cfg.provider, cfg.secret, cfg.ignoredUsers, cfg.allowedUsers)
	if err != nil {
		return err
	}

	return serve(p, cfg.listenAddress)
}

func main() {
	err := run(os.Args, os.Environ(), os.Stdout, os.Stderr, (*proxy.Proxy).Run)
	switch {
	case err == nil:
	case errors.Is(err, pflag.ErrHelp):
		os.Exit(0)
	case errors.Is(err, errUsage):
		os.Exit(2)
	default:
		log.Fatal(err)
	}
}
