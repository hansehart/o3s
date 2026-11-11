// Command o3s sets up the hosts, secrets and features of an o3s home.
package main

import (
	"context"
	"os"

	"github.com/hansehart/o3s/internal/cli"
)

// version is the version a release build sets.
var version string

func main() {
	cli.Version = version
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}))
}
