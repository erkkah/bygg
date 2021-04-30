package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func (b *bygge) handleCopy(target string, cmd string, args ...string) error {
	source := strings.TrimPrefix(cmd, "copy:")
	source = strings.TrimSpace(source)
	if len(source) == 0 {
		if len(args) == 0 {
			return fmt.Errorf("Nothing to copy")
		}
		source = strings.TrimSpace(args[0])
	}
	stat, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("Failed to read source: %w", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("Source must be a file")
	}
	sourceStream, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Failed to open file: %w", err)
	}
	targetStream, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("Failed to create file: %w", err)
	}
	_, err = io.Copy(targetStream, sourceStream)
	if err != nil {
		return err
	}
	return nil
}
