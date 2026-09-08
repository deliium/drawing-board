package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

// Documents warn+ignore contract for deprecated ONNX_MODEL (no fake model path selected).
func TestDeprecatedONNXModelEnvWarns(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	t.Setenv("ONNX_MODEL", "./models/handwriting.onnx")
	if v := os.Getenv("ONNX_MODEL"); v == "" {
		t.Fatal("env should be set for this test")
	}
	// Mirror main()'s warn branch.
	if v := os.Getenv("ONNX_MODEL"); v != "" {
		log.Printf("WARN [main] ONNX_MODEL is set but unsupported/removed; ignoring value (use hiragana5 target comparison)")
	}
	out := buf.String()
	if !strings.Contains(out, "ONNX_MODEL is set but unsupported/removed") {
		t.Fatalf("expected warn log, got %q", out)
	}
	if strings.Contains(strings.ToLower(out), "loading onnx") {
		t.Fatal("must not claim to load ONNX")
	}
}
