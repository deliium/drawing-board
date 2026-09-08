package recognize

import (
	"fmt"

	"github.com/deliium/drawing-board/internal/limits"
)

// validateRecognizeParams enforces canvas bounds and topN (0 means default for non-HTTP callers).
func validateRecognizeParams(width, height, topN int) (int, error) {
	if err := limits.CheckCanvas(width, height); err != nil {
		return 0, err
	}
	if topN == 0 {
		return limits.DefaultTopN, nil
	}
	if topN < limits.MinTopN || topN > limits.MaxTopN {
		return 0, fmt.Errorf("%w: topN out of range", limits.ErrInvalidTopN)
	}
	return topN, nil
}
