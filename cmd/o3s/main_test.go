package main

import (
	"flag"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// update rewrites the expected files inside the scripts to what the binary now writes.
var update = flag.Bool("update", false, "rewrite the expected files inside the scripts")

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
			return nil
		},
	})
}
