package ws

import (
	"errors"
	"math"
	"testing"

	"github.com/deliium/drawing-board/internal/limits"
)

func TestValidateStrokeForTest_OK(t *testing.T) {
	s := &Stroke{
		Points:  []Point{{X: 10, Y: 20}, {X: 30, Y: 20}},
		Color:   "#1d4ed8",
		Width:   4,
		ClientID: "abc",
	}
	if err := ValidateStrokeForTest(s); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateStrokeForTest_Rejects(t *testing.T) {
	cases := []struct {
		name string
		s    *Stroke
		want error
	}{
		{"nil", nil, limits.ErrInvalidStroke},
		{"no points", &Stroke{Color: "#000", Width: 2}, limits.ErrInvalidStroke},
		{"too many points", &Stroke{
			Points: func() []Point {
				pts := make([]Point, limits.MaxPointsPerStroke+1)
				for i := range pts {
					pts[i] = Point{X: 1, Y: 1}
				}
				return pts
			}(),
			Color: "#000", Width: 2,
		}, limits.ErrTooManyPoints},
		{"nan", &Stroke{Points: []Point{{X: math.NaN(), Y: 1}}, Color: "#000", Width: 2}, limits.ErrInvalidCoordinates},
		{"bad width", &Stroke{Points: []Point{{X: 1, Y: 1}}, Color: "#000", Width: 99}, limits.ErrInvalidStroke},
		{"bad color", &Stroke{Points: []Point{{X: 1, Y: 1}}, Color: "blue", Width: 2}, limits.ErrInvalidStroke},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStrokeForTest(tc.s)
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v got %v", tc.want, err)
			}
		})
	}
}

func TestMaxWSMessageBytes(t *testing.T) {
	if limits.MaxWSMessageBytes > 1<<20 {
		t.Fatalf("WS limit must be tighter than old 1MiB, got %d", limits.MaxWSMessageBytes)
	}
	if limits.MaxWSMessageBytes != 64*1024 {
		t.Fatalf("expected 64KiB, got %d", limits.MaxWSMessageBytes)
	}
}
