package recognize

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var debugOn bool

// ConfigureDebug evaluates RECOGNIZE_DEBUG against APP_ENV. Production always disables diagnostics.
func ConfigureDebug(appEnv, recognizeDebug string) {
	prod := strings.EqualFold(strings.TrimSpace(appEnv), "production")
	want := isTruthy(recognizeDebug)
	if prod && want {
		log.Printf("WARN [recognize] RECOGNIZE_DEBUG ignored in production")
		debugOn = false
		log.Printf("INFO [recognize] recognize_debug=false")
		return
	}
	debugOn = want && !prod
	log.Printf("INFO [recognize] recognize_debug=%t", debugOn)
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// DebugEnabled reports whether handwriting diagnostics may be emitted.
func DebugEnabled() bool {
	return debugOn
}

// ResetDebugForTest resets the debug gate (tests only).
func ResetDebugForTest() {
	debugOn = false
}

// debugf emits detailed handwriting diagnostics only when ConfigureDebug enabled them.
func debugf(format string, args ...interface{}) {
	if !debugOn {
		return
	}
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "error", "warn", "info":
		return
	}
	msg := fmt.Sprintf(format, args...)
	log.Printf("DEBUG [recognize] %s", msg)
}
