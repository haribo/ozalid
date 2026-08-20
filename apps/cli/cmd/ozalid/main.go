// Command ozalid pushes editions to an ozalid server.
//
// The runner adapter lives here, never on the server (ADR 0003).
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ozalid <command>")
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
	os.Exit(2)
}
