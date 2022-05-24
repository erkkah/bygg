package main

import (
	"regexp"
	"runtime/debug"
)

// FallbackTag is used when there is no version in build info
var FallbackTag = "v0.6.0"

func BuildTag() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		version := info.Main.Version
		parser := regexp.MustCompile(`^(v\d+[.]\d+.\d+).*`)
		match := parser.FindSubmatch([]byte(version))
		if match != nil {
			return string(match[1])
		}
	}
	return FallbackTag
}
