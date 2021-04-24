package main

import (
	"fmt"
	"os"
	"strings"
)

func (b *bygge) handleMakeDir(cmd string) error {
	path := strings.TrimPrefix(cmd, "mkdir:")
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
