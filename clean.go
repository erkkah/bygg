package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func (b *bygge) handleClean(cmd string, args ...string) error {
	path := strings.TrimPrefix(cmd, "clean:")
	stat, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if stat == nil {
		return nil
	}
	recursive := len(args) > 0 && args[0] == "-r"
	if stat.IsDir() && !recursive {
		return fmt.Errorf("%q is a directory and \"-r\" was not specified", path)
	}
	err = os.RemoveAll(path)
	if err != nil {
		return err
	}
	return nil
}
