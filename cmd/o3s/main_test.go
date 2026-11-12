package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/hansehart/o3s/internal/registry/registrytest"
)

// update rewrites the expected files inside the scripts to what the binary now writes.
var update = flag.Bool("update", false, "rewrite the expected files inside the scripts")

// fakeDocker stands in for the container engine, so a script sees which restarts a change asks for.
const fakeDocker = `#!/bin/sh
echo "$*" >> "$WORK/docker.log"
# Answer each question about the stack, and any restart, the way a host's engine would
case "$*" in
  ps*service=gateway*) [ -f "$WORK/stack-up" ] && echo gateway-id ;;
  ps*service=proxy*) [ -f "$WORK/stack-up" ] && echo proxy-id ;;
  restart*) echo "$2" ;;
esac
exit 0
`

// fakeDevcontainer stands in for the devcontainer CLI, so a script sees each rebuild an update asks for.
const fakeDevcontainer = `#!/bin/sh
echo "$*" >> "$WORK/devcontainer.log"
`

// tool is a feature that needs a host with a secret, sets env, and keeps its data in a state dir.
const tool = `{
  "id": "tool",
  "version": "2.1.0",
  "options": {"stateDir": {"type": "string", "default": ""}},
  "customizations": {"o3s": {
    "hosts": [{"name": "example.com", "ports": [443], "secret": "EXAMPLE_TOKEN", "description": "API"}],
    "env": [
      {"name": "EXAMPLE_MODE", "value": "strict", "description": "Mode"},
      {"name": "EXAMPLE_HOME", "value": "${stateDir}", "description": "Data"}
    ]
  }}
}`

// TestMain lets the scripts run this binary as o3s.
func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"o3s": main})
}

// Every scenario in the scripts holds for the o3s binary.
func TestScripts(t *testing.T) {
	reg, err := registrytest.New()
	// Stop when the registry fails to come up
	if err != nil {
		t.Fatalf("Setup: registrytest.New() = %v", err)
	}
	t.Cleanup(reg.Close)
	// Stop when the store fails to publish
	if err := reg.PushFeatures("features", `{"id": "node"}`, `{"id": "go", "version": "1.4.0"}`, tool); err != nil {
		t.Fatalf("Setup: PushFeatures(features) = %v", err)
	}
	testscript.Run(t, testscript.Params{
		Dir:                 "testdata/script",
		RequireExplicitExec: true,
		UpdateScripts:       *update,
		Setup: func(env *testscript.Env) error {
			env.Setenv("HOME", env.WorkDir)
			env.Setenv("REGISTRY", reg.Host)
			bin := filepath.Join(env.WorkDir, ".bin")
			// Hand back a fake engine that stays unwritten
			if err := os.MkdirAll(bin, 0o750); err != nil {
				return err
			}
			// Hand back a fake engine that stays unwritten
			if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(fakeDocker), 0o755); err != nil { //nolint:gosec // the scripts run the fake engine
				return err
			}
			// Hand back a fake devcontainer CLI that stays unwritten
			if err := os.WriteFile(filepath.Join(bin, "devcontainer"), []byte(fakeDevcontainer), 0o755); err != nil { //nolint:gosec // the scripts run the fake CLI
				return err
			}
			env.Setenv("PATH", bin+string(os.PathListSeparator)+env.Getenv("PATH"))
			return nil
		},
	})
}
