package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// update rewrites the expected files inside the scripts to what the binary now writes.
var update = flag.Bool("update", false, "rewrite the expected files inside the scripts")

// fakeDocker records every call and answers like a host whose stack runs while the script's folder
// holds a stack-up file.
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

// TestMain lets the scripts run this binary as o3s.
func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"o3s": main})
}

// Every scenario in the scripts holds for the o3s binary.
func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:                 "testdata/script",
		RequireExplicitExec: true,
		UpdateScripts:       *update,
		Setup: func(env *testscript.Env) error {
			env.Setenv("HOME", env.WorkDir)
			bin := filepath.Join(env.WorkDir, ".bin")
			// Hand back a fake engine that stays unwritten
			if err := os.MkdirAll(bin, 0o750); err != nil {
				return err
			}
			// Hand back a fake engine that stays unwritten
			if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(fakeDocker), 0o755); err != nil { //nolint:gosec // the scripts run the fake engine
				return err
			}
			env.Setenv("PATH", bin+string(os.PathListSeparator)+env.Getenv("PATH"))
			return nil
		},
	})
}
