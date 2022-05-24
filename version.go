package main

import (
	"fmt"
	"runtime/debug"
)

// Tag is set to the current git tag, or empty for dev version
// ??? Update this when / if this makes it into a release: https://github.com/golang/go/issues/49168
var Tag = "v0.6.0"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("Build info: %v\n", info)
	}
}
