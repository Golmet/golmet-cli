// Command golmet é o entrypoint do CLI do kit GOLMET.
package main

import (
	"fmt"
	"os"

	"github.com/Golmet/golmet-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		if code, ok := cli.ExitCodeOf(err); ok {
			os.Exit(code)
		}
		fmt.Fprintln(os.Stderr, "golmet:", err)
		os.Exit(2)
	}
}
