// "bygg" is an attempt to replace the roles of "make" and "bash" in building
// go projects, making it easier to maintain a portable build environment.
package main

import (
	"fmt"
	"os"
)

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	b, err := newBygge(cfg)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	b.verbose("Building target %q", cfg.target)
	err = b.buildTarget(cfg.target)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
