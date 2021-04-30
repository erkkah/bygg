package main

import (
	"fmt"
	"os"
	"strings"
)

func (b *bygge) handleMakeDir(target string, cmd string, args ...string) error {
	path := strings.TrimPrefix(cmd, "mkdir:")
	path = strings.TrimSpace(path)
	if len(path) == 0 {
		if len(args) == 0 {
			path = target
		}
		path = strings.TrimSpace(args[0])
	}
	stat, err := os.Stat(path)
	if err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("Will not overwrite non-dir target %q", path)
		}
		return nil
	}
	err = os.MkdirAll(path, 0771)
	if err != nil {
		return err
	}
	return nil
}
