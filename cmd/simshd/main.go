package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/service/httpapi"
)

type launcherDeps struct {
	getwd      func() (string, error)
	newHandler func(httpapi.Config) http.Handler
	serve      func(string, http.Handler) error
	logf       func(string, ...any)
}

type launchConfig struct {
	listenAddr string
	handler    httpapi.Config
}

func main() {
	if err := run(os.Args[1:], launcherDeps{
		getwd:      os.Getwd,
		newHandler: httpapi.NewHandler,
		serve:      http.ListenAndServe,
		logf:       log.Printf,
	}); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, deps launcherDeps) error {
	if deps.getwd == nil {
		deps.getwd = os.Getwd
	}
	if deps.newHandler == nil {
		deps.newHandler = httpapi.NewHandler
	}
	if deps.serve == nil {
		deps.serve = http.ListenAndServe
	}
	if deps.logf == nil {
		deps.logf = log.Printf
	}

	cfg, err := parseLaunchConfig(args, deps.getwd)
	if err != nil {
		return err
	}

	deps.logf(
		"simshd listening on %s host_root=%s profile=%s policy=%s test_mount=%v",
		cfg.listenAddr,
		cfg.handler.DefaultHostRoot,
		cfg.handler.DefaultProfile,
		cfg.handler.DefaultPolicy,
		cfg.handler.EnableTestMount,
	)
	handler := deps.newHandler(cfg.handler)
	if err := deps.serve(cfg.listenAddr, handler); err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	return nil
}

func parseLaunchConfig(args []string, getwd func() (string, error)) (launchConfig, error) {
	fs := flag.NewFlagSet("simshd", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	listen := fs.String("listen", "127.0.0.1:18080", "http listen address")
	port := fs.Int("P", 0, "port override for web runtime service")
	rootDir := fs.String("root", "", "default root dir for execute")
	profile := fs.String("profile", string(contract.ProfileCoreStrict), "default profile")
	policy := fs.String("policy", string(contract.WriteModeReadOnly), "default policy")
	enableTestMount := fs.Bool("enable-test-mount", false, "enable /test regression corpus mount")
	if err := fs.Parse(args); err != nil {
		return launchConfig{}, err
	}

	root := strings.TrimSpace(*rootDir)
	if root == "" && getwd != nil {
		wd, err := getwd()
		if err == nil {
			root = wd
		}
	}

	profilePreset, err := contract.ParseProfile(*profile)
	if err != nil {
		return launchConfig{}, fmt.Errorf("invalid profile: %w", err)
	}
	policyPreset, err := contract.PolicyPreset(*policy)
	if err != nil {
		return launchConfig{}, fmt.Errorf("invalid policy: %w", err)
	}

	listenAddr := strings.TrimSpace(*listen)
	if *port > 0 {
		listenAddr = fmt.Sprintf("127.0.0.1:%d", *port)
	}

	return launchConfig{
		listenAddr: listenAddr,
		handler: httpapi.Config{
			DefaultHostRoot: root,
			DefaultProfile:  string(profilePreset),
			DefaultPolicy:   string(policyPreset.WriteMode),
			EnableTestMount: *enableTestMount,
		},
	}, nil
}
