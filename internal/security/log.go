package security

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// secLogf emits level-prefixed lines. DEBUG is suppressed when LOG_LEVEL=info|warn|error.
func secLogf(level, format string, args ...interface{}) {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "error":
		if level == "DEBUG" || level == "INFO" || level == "WARN" {
			return
		}
	case "warn":
		if level == "DEBUG" || level == "INFO" {
			return
		}
	case "info":
		if level == "DEBUG" {
			return
		}
	}
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s %s", level, msg)
}
