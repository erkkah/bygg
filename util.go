package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func splitQuoted(quoted string) ([]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(quoted))
	scanner.Split(bufio.ScanRunes)

	parts := []string{}

	escapeNext := false
	inString := false

	var builder strings.Builder

	for scanner.Scan() {
		char := scanner.Text()
		switch char {
		case `\`:
			if inString {
				escapeNext = true
			} else {
				builder.WriteString(char)
			}
		case `"`:
			if escapeNext {
				builder.WriteString(char)
			} else {
				inString = !inString
			}
			escapeNext = false
		case ` `:
			if inString {
				builder.WriteString(char)
			} else if builder.Len() != 0 {
				parts = append(parts, builder.String())
				builder.Reset()
			}
			escapeNext = false
		default:
			if escapeNext && char == "n" {
				char = "\n"
			}
			builder.WriteString(char)
			escapeNext = false
		}
	}
	if inString {
		return parts, fmt.Errorf("unterminated string")
	}
	if builder.Len() != 0 {
		parts = append(parts, builder.String())
	}
	return parts, nil
}

func exists(target string) bool {
	stat, err := os.Stat(target)
	return err == nil && stat != nil
}

func getFileDate(target string) time.Time {
	fileInfo, _ := os.Stat(target)
	if fileInfo == nil {
		return time.Time{}
	}
	return fileInfo.ModTime()
}
