// Command yoink is an encrypted macOS clipboard-history tool. See
// `yoink help` for usage. This entry point stays thin: all behavior lives in
// internal/cli.
package main

import (
	"fmt"
	"os"

	"github.com/davcevski/yoink/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "yoink: "+err.Error())
		os.Exit(1)
	}
}
