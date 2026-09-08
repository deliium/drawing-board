// Package limits holds shared input bounds for recognition and stroke ingestion.
package limits

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
)

// Limit contract (Prompt 04). Tune only with tests + README.
const (
	MaxRecognizeBodyBytes = 4 * 1024
	MaxAPIJSONBodyBytes   = 64 * 1024
	MaxWSMessageBytes     = 64 * 1024

	MinCanvasDim    = 1
	MaxCanvasDim    = 2048
	MaxCanvasPixels = 2048 * 2048 // 2_097_152

	MinTopN     = 1
	MaxTopN     = 32
	DefaultTopN = 10

	MaxStrokesPerRecognize = 64
	MaxPointsPerStroke     = 2048
	MaxPointsPerRecognize  = 16_384

	MinCoord = -512
	MaxCoord = 4096

	MinStrokeWidth = 1
	MaxStrokeWidth = 20

	MaxColorLen    = 32
	MaxClientIDLen = 64

	RecognizeRatePerMin = 30
	RecognizeBurst      = 5
	StrokeIngestPerMin  = 60
	StrokeIngestBurst   = 20
)

var (
	ErrInvalidDimensions = errors.New("invalid_dimensions")
	ErrInvalidTopN       = errors.New("invalid_top_n")
	ErrTooManyStrokes    = errors.New("too_many_strokes")
	ErrTooManyPoints     = errors.New("too_many_points")
	ErrInvalidStrokeData = errors.New("invalid_stroke_data")
	ErrInvalidStroke     = errors.New("invalid_stroke")
	ErrInvalidCoordinates = errors.New("invalid_coordinates")
)

var colorRe = regexp.MustCompile(`(?i)^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$`)

// PointXY is implemented by stroke point types across packages.
type PointXY interface {
	GetX() float64
	GetY() float64
}

// FloatPoint adapts raw float coordinates.
type FloatPoint struct {
	X, Y float64
}

func (p FloatPoint) GetX() float64 { return p.X }
func (p FloatPoint) GetY() float64 { return p.Y }

// CheckCanvas validates width/height and the pixel product without overflowing.
func CheckCanvas(width, height int) error {
	if width < MinCanvasDim || width > MaxCanvasDim || height < MinCanvasDim || height > MaxCanvasDim {
		return fmt.Errorf("%w: width/height out of range", ErrInvalidDimensions)
	}
	if uint64(width)*uint64(height) > uint64(MaxCanvasPixels) {
		return fmt.Errorf("%w: pixel product too large", ErrInvalidDimensions)
	}
	return nil
}

// NormalizeOrValidateTopN defaults omitted topN to DefaultTopN; rejects invalid present values.
func NormalizeOrValidateTopN(topN int, present bool) (int, error) {
	if !present {
		return DefaultTopN, nil
	}
	if topN < MinTopN || topN > MaxTopN {
		return 0, fmt.Errorf("%w: topN must be %d..%d", ErrInvalidTopN, MinTopN, MaxTopN)
	}
	return topN, nil
}

// ValidateStrokeMeta checks line width, color, and clientId.
func ValidateStrokeMeta(width int, color, clientID string) error {
	if width < MinStrokeWidth || width > MaxStrokeWidth {
		return fmt.Errorf("%w: width must be %d..%d", ErrInvalidStroke, MinStrokeWidth, MaxStrokeWidth)
	}
	if err := validateColor(color); err != nil {
		return err
	}
	if len(clientID) > MaxClientIDLen {
		return fmt.Errorf("%w: clientId too long", ErrInvalidStroke)
	}
	for _, r := range clientID {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: clientId has control chars", ErrInvalidStroke)
		}
	}
	return nil
}

func validateColor(color string) error {
	if color == "" {
		return fmt.Errorf("%w: color required", ErrInvalidStroke)
	}
	if len(color) > MaxColorLen {
		return fmt.Errorf("%w: color too long", ErrInvalidStroke)
	}
	for _, r := range color {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: color has control chars", ErrInvalidStroke)
		}
	}
	if !colorRe.MatchString(strings.TrimSpace(color)) {
		return fmt.Errorf("%w: color must be #RGB, #RRGGBB, or #RRGGBBAA", ErrInvalidStroke)
	}
	return nil
}

// ValidateCoord rejects NaN/Inf and out-of-range coordinates.
func ValidateCoord(x, y float64) error {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return fmt.Errorf("%w: NaN/Inf", ErrInvalidCoordinates)
	}
	if x < MinCoord || x > MaxCoord || y < MinCoord || y > MaxCoord {
		return fmt.Errorf("%w: out of range", ErrInvalidCoordinates)
	}
	return nil
}

// ValidateStrokePoints checks point count and each coordinate.
func ValidateStrokePoints(points []FloatPoint) error {
	if len(points) == 0 {
		return fmt.Errorf("%w: points required", ErrInvalidStroke)
	}
	if len(points) > MaxPointsPerStroke {
		return fmt.Errorf("%w: points per stroke", ErrTooManyPoints)
	}
	for _, p := range points {
		if err := ValidateCoord(p.X, p.Y); err != nil {
			return err
		}
	}
	return nil
}

// ValidateStrokeSet checks aggregate stroke/point caps for a recognition attempt.
func ValidateStrokeSet(strokeCount, totalPoints int) error {
	if strokeCount > MaxStrokesPerRecognize {
		return fmt.Errorf("%w", ErrTooManyStrokes)
	}
	if totalPoints > MaxPointsPerRecognize {
		return fmt.Errorf("%w: total points", ErrTooManyPoints)
	}
	return nil
}

// ErrorCode returns the stable API/WS error code for a limits error.
func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidDimensions):
		return "invalid_dimensions"
	case errors.Is(err, ErrInvalidTopN):
		return "invalid_top_n"
	case errors.Is(err, ErrTooManyStrokes):
		return "too_many_strokes"
	case errors.Is(err, ErrTooManyPoints):
		return "too_many_points"
	case errors.Is(err, ErrInvalidStrokeData):
		return "invalid_stroke_data"
	case errors.Is(err, ErrInvalidStroke):
		return "invalid_stroke"
	case errors.Is(err, ErrInvalidCoordinates):
		return "invalid_coordinates"
	default:
		return "invalid_stroke"
	}
}

// SafeMessage returns a short client-safe message for a limits error.
func SafeMessage(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, ErrInvalidDimensions):
		return "canvas width/height out of allowed range"
	case errors.Is(err, ErrInvalidTopN):
		return "topN out of allowed range"
	case errors.Is(err, ErrTooManyStrokes):
		return "too many strokes for recognition"
	case errors.Is(err, ErrTooManyPoints):
		return "too many points"
	case errors.Is(err, ErrInvalidStrokeData):
		return "stored stroke data is invalid"
	case errors.Is(err, ErrInvalidCoordinates):
		return "coordinates out of range"
	case errors.Is(err, ErrInvalidStroke):
		return "invalid stroke"
	default:
		return "invalid input"
	}
}
