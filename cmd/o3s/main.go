// Command o3s sets up the cage an o3s home runs: the hosts it may reach, the secrets the proxy
// injects, and the features it holds.
package main

import (
	"context"
	"os"

	"github.com/hansehart/o3s/internal/cli"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}))
}
