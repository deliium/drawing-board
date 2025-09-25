package recognize

import (
	"math"
)

// SimpleRecognizer provides basic stroke pattern matching without external dependencies
type SimpleRecognizer struct{}

func NewSimpleRecognizer() *SimpleRecognizer {
	return &SimpleRecognizer{}
}

func (s *SimpleRecognizer) Close() error {
	return nil
}

// analyzeStrokeDirection determines the primary direction of a stroke
func analyzeStrokeDirection(stroke Stroke) string {
	if len(stroke.Points) < 2 {
		return "dot"
	}
	
	start := stroke.Points[0]
	end := stroke.Points[len(stroke.Points)-1]
	
	dx := end.X - start.X
	dy := end.Y - start.Y
	
	// Calculate angle in degrees
	angle := math.Atan2(dy, dx) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	
	// Classify direction
	if math.Abs(dx) < 5 && math.Abs(dy) < 5 {
		return "dot"
	} else if angle >= 315 || angle < 45 || angle >= 135 && angle < 225 {
		return "horizontal"
	} else if angle >= 45 && angle < 135 || angle >= 225 && angle < 315 {
		return "vertical"
	} else if angle >= 45 && angle < 135 {
		return "diagonal_up"
	} else {
		return "diagonal_down"
	}
}

// analyzeStrokeShape determines if a stroke is straight, curved, or complex
func analyzeStrokeShape(stroke Stroke) string {
	if len(stroke.Points) < 3 {
		return "straight"
	}
	
	// Calculate total deviation from straight line
	totalDeviation := 0.0
	start := stroke.Points[0]
	end := stroke.Points[len(stroke.Points)-1]
	
	for i := 1; i < len(stroke.Points)-1; i++ {
		point := stroke.Points[i]
		// Distance from point to line between start and end
		deviation := math.Abs((end.Y-start.Y)*point.X - (end.X-start.X)*point.Y + end.X*start.Y - end.Y*start.X) / 
			math.Sqrt(math.Pow(end.Y-start.Y, 2) + math.Pow(end.X-start.X, 2))
		totalDeviation += deviation
	}
	
	avgDeviation := totalDeviation / float64(len(stroke.Points)-2)
	
	if avgDeviation < 5 {
		return "straight"
	} else if avgDeviation < 15 {
		return "slightly_curved"
	} else {
		return "curved"
	}
}

// Simple pattern matching based on stroke count and basic shape analysis
func (s *SimpleRecognizer) Recognize(strokes []Stroke, width, height int, topN int) ([]Candidate, error) {
	if topN <= 0 {
		topN = 10
	}
	
	if len(strokes) == 0 {
		return []Candidate{}, nil
	}
	
	// Analyze stroke patterns
	totalPoints := 0
	strokeDirections := make([]string, len(strokes))
	strokeShapes := make([]string, len(strokes))
	
	for i, stroke := range strokes {
		totalPoints += len(stroke.Points)
		strokeDirections[i] = analyzeStrokeDirection(stroke)
		strokeShapes[i] = analyzeStrokeShape(stroke)
	}
	
	candidates := []Candidate{}
	
	// Single stroke analysis
	if len(strokes) == 1 {
		dir := strokeDirections[0]
		shape := strokeShapes[0]
		
		if dir == "horizontal" && shape == "straight" {
			candidates = append(candidates, 
				Candidate{Text: "一", Score: 0.9}, // horizontal line
				Candidate{Text: "ー", Score: 0.7}, // long vowel mark
			)
		} else if dir == "vertical" && shape == "straight" {
			candidates = append(candidates,
				Candidate{Text: "丨", Score: 0.9}, // vertical line
				Candidate{Text: "｜", Score: 0.7}, // vertical bar
			)
		} else if dir == "dot" {
			candidates = append(candidates,
				Candidate{Text: "丶", Score: 0.8}, // dot
				Candidate{Text: "。", Score: 0.6}, // period
			)
		} else if shape == "curved" {
			candidates = append(candidates,
				Candidate{Text: "し", Score: 0.7}, // curved stroke
				Candidate{Text: "く", Score: 0.5}, // curved stroke
			)
		}
	}
	
	// Two stroke analysis
	if len(strokes) == 2 {
		dir1, dir2 := strokeDirections[0], strokeDirections[1]
		
		if dir1 == "horizontal" && dir2 == "horizontal" {
			candidates = append(candidates,
				Candidate{Text: "二", Score: 0.8}, // two horizontal lines
				Candidate{Text: "ニ", Score: 0.6}, // katakana ni
			)
		} else if (dir1 == "horizontal" && dir2 == "vertical") || (dir1 == "vertical" && dir2 == "horizontal") {
			candidates = append(candidates,
				Candidate{Text: "十", Score: 0.8}, // cross
				Candidate{Text: "＋", Score: 0.6}, // plus
			)
		} else if dir1 == "diagonal_up" && dir2 == "diagonal_up" {
			candidates = append(candidates,
				Candidate{Text: "人", Score: 0.7}, // person
				Candidate{Text: "入", Score: 0.5}, // enter
			)
		}
	}
	
	// Three stroke analysis
	if len(strokes) == 3 {
		dir1, dir2, dir3 := strokeDirections[0], strokeDirections[1], strokeDirections[2]
		
		if dir1 == "horizontal" && dir2 == "horizontal" && dir3 == "horizontal" {
			candidates = append(candidates,
				Candidate{Text: "三", Score: 0.8}, // three horizontal lines
				Candidate{Text: "ミ", Score: 0.6}, // katakana mi
			)
		} else if dir1 == "horizontal" && dir2 == "vertical" && dir3 == "diagonal_down" {
			candidates = append(candidates,
				Candidate{Text: "大", Score: 0.7}, // big
				Candidate{Text: "太", Score: 0.5}, // fat
			)
		}
	}
	
	// Complex characters (4+ strokes)
	if len(strokes) >= 4 {
		// Analyze complexity patterns
		horizontalCount := 0
		verticalCount := 0
		for _, dir := range strokeDirections {
			if dir == "horizontal" {
				horizontalCount++
			} else if dir == "vertical" {
				verticalCount++
			}
		}
		
		if horizontalCount >= 2 && verticalCount >= 2 {
			candidates = append(candidates,
				Candidate{Text: "中", Score: 0.6}, // middle
				Candidate{Text: "田", Score: 0.5}, // field
			)
		}
		
		candidates = append(candidates,
			Candidate{Text: "国", Score: 0.5}, // country
			Candidate{Text: "学", Score: 0.4}, // study
			Candidate{Text: "生", Score: 0.3}, // life
		)
	}
	
	// Add complexity-based characters
	if totalPoints > 20 {
		candidates = append(candidates,
			Candidate{Text: "書", Score: 0.3}, // write
			Candidate{Text: "字", Score: 0.2}, // character
		)
	}
	
	// If no specific matches, provide generic suggestions based on stroke count
	if len(candidates) == 0 {
		if len(strokes) == 1 {
			candidates = append(candidates, Candidate{Text: "一", Score: 0.5})
		} else if len(strokes) == 2 {
			candidates = append(candidates, Candidate{Text: "二", Score: 0.5})
		} else if len(strokes) == 3 {
			candidates = append(candidates, Candidate{Text: "三", Score: 0.5})
		} else {
			candidates = append(candidates, Candidate{Text: "中", Score: 0.4})
		}
	}
	
	// Limit to topN results
	if len(candidates) > topN {
		candidates = candidates[:topN]
	}
	
	return candidates, nil
}
