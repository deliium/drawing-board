package limits

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestCheckCanvas(t *testing.T) {
	cases := []struct {
		name    string
		w, h    int
		wantErr bool
	}{
		{"min", 1, 1, false},
		{"max square", MaxCanvasDim, MaxCanvasDim, false},
		{"typical", 300, 300, false},
		{"zero w", 0, 100, true},
		{"zero h", 100, 0, true},
		{"neg", -1, 100, true},
		{"over dim", MaxCanvasDim + 1, 1, true},
		{"over height", 1, MaxCanvasDim + 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckCanvas(tc.w, tc.h)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr && !errors.Is(err, ErrInvalidDimensions) {
				t.Fatalf("want ErrInvalidDimensions, got %v", err)
			}
		})
	}
}

func TestCheckCanvas_OverflowPair(t *testing.T) {
	// Both dims within MaxCanvasDim but product would overflow int32-style misuse;
	// MaxCanvasPixels == MaxCanvasDim^2 so equal is OK.
	if err := CheckCanvas(MaxCanvasDim, MaxCanvasDim); err != nil {
		t.Fatalf("max square should be ok: %v", err)
	}
}

func TestNormalizeOrValidateTopN(t *testing.T) {
	n, err := NormalizeOrValidateTopN(0, false)
	if err != nil || n != DefaultTopN {
		t.Fatalf("omitted → default: got %d %v", n, err)
	}
	n, err = NormalizeOrValidateTopN(10, true)
	if err != nil || n != 10 {
		t.Fatalf("valid present: got %d %v", n, err)
	}
	for _, bad := range []int{0, -1, MaxTopN + 1} {
		_, err := NormalizeOrValidateTopN(bad, true)
		if !errors.Is(err, ErrInvalidTopN) {
			t.Fatalf("present %d: want ErrInvalidTopN, got %v", bad, err)
		}
	}
	n, err = NormalizeOrValidateTopN(MaxTopN, true)
	if err != nil || n != MaxTopN {
		t.Fatalf("max topN: got %d %v", n, err)
	}
}

func TestValidateStrokeMeta(t *testing.T) {
	if err := ValidateStrokeMeta(4, "#1d4ed8", "abc"); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if err := ValidateStrokeMeta(0, "#fff", "a"); !errors.Is(err, ErrInvalidStroke) {
		t.Fatalf("bad width: %v", err)
	}
	if err := ValidateStrokeMeta(21, "#fff", "a"); !errors.Is(err, ErrInvalidStroke) {
		t.Fatalf("wide width: %v", err)
	}
	if err := ValidateStrokeMeta(4, "red", "a"); !errors.Is(err, ErrInvalidStroke) {
		t.Fatalf("named color: %v", err)
	}
	if err := ValidateStrokeMeta(4, "#gg0000", "a"); !errors.Is(err, ErrInvalidStroke) {
		t.Fatalf("bad hex: %v", err)
	}
	longID := strings.Repeat("a", MaxClientIDLen+1)
	if err := ValidateStrokeMeta(4, "#000", longID); !errors.Is(err, ErrInvalidStroke) {
		t.Fatalf("long clientId: %v", err)
	}
}

func TestValidateStrokePoints(t *testing.T) {
	if err := ValidateStrokePoints(nil); !errors.Is(err, ErrInvalidStroke) {
		t.Fatalf("empty: %v", err)
	}
	if err := ValidateStrokePoints([]FloatPoint{{X: 1, Y: 2}}); err != nil {
		t.Fatalf("ok: %v", err)
	}
	if err := ValidateStrokePoints([]FloatPoint{{X: math.NaN(), Y: 1}}); !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatalf("nan: %v", err)
	}
	if err := ValidateStrokePoints([]FloatPoint{{X: math.Inf(1), Y: 1}}); !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatalf("inf: %v", err)
	}
	if err := ValidateStrokePoints([]FloatPoint{{X: MaxCoord + 1, Y: 0}}); !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatalf("range: %v", err)
	}
	tooMany := make([]FloatPoint, MaxPointsPerStroke+1)
	for i := range tooMany {
		tooMany[i] = FloatPoint{X: 1, Y: 1}
	}
	if err := ValidateStrokePoints(tooMany); !errors.Is(err, ErrTooManyPoints) {
		t.Fatalf("too many: %v", err)
	}
}

func TestValidateStrokeSet(t *testing.T) {
	if err := ValidateStrokeSet(MaxStrokesPerRecognize, MaxPointsPerRecognize); err != nil {
		t.Fatalf("at limit: %v", err)
	}
	if err := ValidateStrokeSet(MaxStrokesPerRecognize+1, 1); !errors.Is(err, ErrTooManyStrokes) {
		t.Fatalf("strokes: %v", err)
	}
	if err := ValidateStrokeSet(1, MaxPointsPerRecognize+1); !errors.Is(err, ErrTooManyPoints) {
		t.Fatalf("points: %v", err)
	}
}

func TestErrorCodeAndMessage(t *testing.T) {
	if ErrorCode(ErrInvalidDimensions) != "invalid_dimensions" {
		t.Fatal(ErrorCode(ErrInvalidDimensions))
	}
	if SafeMessage(ErrTooManyStrokes) == "" {
		t.Fatal("empty message")
	}
}
