package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/service/httpapi"
)

func main() {
	listen := flag.String("listen", ":18080", "http listen address")
	port := flag.Int("P", 0, "port override for web runtime service")
	rootDir := flag.String("root", "", "default root dir for execute")
	profile := flag.String("profile", string(contract.ProfileCoreStrict), "default profile")
	policy := flag.String("policy", string(contract.WriteModeReadOnly), "default policy")
	enableTestMount := flag.Bool("enable-test-mount", false, "enable /test regression corpus mount")
	flag.Parse()

	if strings.TrimSpace(*rootDir) == "" {
		wd, err := os.Getwd()
		if err == nil {
			*rootDir = wd
		}
	}

	profilePreset, err := contract.ParseProfile(*profile)
	if err != nil {
		log.Fatalf("invalid profile: %v", err)
	}
	policyPreset, err := contract.PolicyPreset(*policy)
	if err != nil {
		log.Fatalf("invalid policy: %v", err)
	}

	listenAddr := strings.TrimSpace(*listen)
	if *port > 0 {
		listenAddr = fmt.Sprintf(":%d", *port)
	}
	log.Printf("simshd listening on %s host_root=%s profile=%s policy=%s test_mount=%v", listenAddr, *rootDir, *profile, *policy, *enableTestMount)
	handler := httpapi.NewHandler(httpapi.Config{
		DefaultHostRoot: *rootDir,
		DefaultProfile:  string(profilePreset),
		DefaultPolicy:   string(policyPreset.WriteMode),
		EnableTestMount: *enableTestMount,
	})
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}
