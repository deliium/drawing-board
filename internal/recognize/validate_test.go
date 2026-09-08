package recognize

import (
	"errors"
	"testing"

	"github.com/deliium/drawing-board/internal/limits"
)

func TestValidateRecognizeParams(t *testing.T) {
	n, err := validateRecognizeParams(300, 300, 10)
	if err != nil || n != 10 {
		t.Fatalf("got %d %v", n, err)
	}
	n, err = validateRecognizeParams(300, 300, 0)
	if err != nil || n != limits.DefaultTopN {
		t.Fatalf("default: %d %v", n, err)
	}
	_, err = validateRecognizeParams(999999, 999999, 10)
	if !errors.Is(err, limits.ErrInvalidDimensions) {
		t.Fatalf("want invalid dims, got %v", err)
	}
	_, err = validateRecognizeParams(300, 300, limits.MaxTopN+1)
	if !errors.Is(err, limits.ErrInvalidTopN) {
		t.Fatalf("want invalid topN, got %v", err)
	}
}

func TestSimpleRecognizer_RejectsHugeCanvas(t *testing.T) {
	r := NewSimpleRecognizer()
	_, err := r.Recognize([]Stroke{{Points: []Point{{X: 1, Y: 1}}}}, 999999, 999999, 10)
	if !errors.Is(err, limits.ErrInvalidDimensions) {
		t.Fatalf("got %v", err)
	}
}

func TestTargetCompare_RejectsHugeCanvas(t *testing.T) {
	r, err := NewTargetCompareRecognizer()
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	_, err = r.Assess("あ", []Stroke{{Points: []Point{{X: 1, Y: 1}}}}, 999999, 999999)
	if !errors.Is(err, limits.ErrInvalidDimensions) {
		t.Fatalf("got %v", err)
	}
}

func TestConfigureDebug_ProductionForcesOff(t *testing.T) {
	ResetDebugForTest()
	ConfigureDebug("production", "1")
	if DebugEnabled() {
		t.Fatal("debug must be off in production")
	}
	ResetDebugForTest()
	ConfigureDebug("development", "1")
	if !DebugEnabled() {
		t.Fatal("debug should be on in non-production")
	}
	ResetDebugForTest()
}
