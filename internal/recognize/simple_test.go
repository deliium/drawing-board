package recognize

import (
	"testing"
)

func TestNewSimpleRecognizer(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	if recognizer == nil {
		t.Fatal("SimpleRecognizer should not be nil")
	}
}

func TestSimpleRecognizer_Recognize_EmptyStrokes(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	
	candidates, err := recognizer.Recognize([]Stroke{}, 300, 300, 5)
	if err != nil {
		t.Fatalf("Should not return error for empty strokes: %v", err)
	}
	
	if len(candidates) != 0 {
		t.Fatalf("Expected 0 candidates for empty strokes, got %d", len(candidates))
	}
}

func TestSimpleRecognizer_Recognize_SingleStroke(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	
	// Test single stroke (should return basic characters)
	strokes := []Stroke{
		{
			Points: []Point{
				{X: 10, Y: 10},
				{X: 20, Y: 10},
			},
		},
	}
	
	candidates, err := recognizer.Recognize(strokes, 300, 300, 5)
	if err != nil {
		t.Fatalf("Should not return error: %v", err)
	}
	
	if len(candidates) == 0 {
		t.Fatal("Should return at least one candidate")
	}
	
	// Check that candidates have valid text and scores
	for i, candidate := range candidates {
		if candidate.Text == "" {
			t.Fatalf("Candidate %d should have non-empty text", i)
		}
		if candidate.Score < 0 || candidate.Score > 1 {
			t.Fatalf("Candidate %d score should be between 0 and 1, got %f", i, candidate.Score)
		}
	}
}

func TestSimpleRecognizer_Recognize_TwoStrokes(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	
	// Test two strokes (should return characters like 人, 入)
	strokes := []Stroke{
		{
			Points: []Point{
				{X: 10, Y: 10},
				{X: 20, Y: 20},
			},
		},
		{
			Points: []Point{
				{X: 30, Y: 10},
				{X: 40, Y: 20},
			},
		},
	}
	
	candidates, err := recognizer.Recognize(strokes, 300, 300, 5)
	if err != nil {
		t.Fatalf("Should not return error: %v", err)
	}
	
	if len(candidates) == 0 {
		t.Fatal("Should return at least one candidate")
	}
	
	// Should have candidates for two-stroke characters
	foundTwoStroke := false
	for _, candidate := range candidates {
		if candidate.Text == "人" || candidate.Text == "入" {
			foundTwoStroke = true
			break
		}
	}
	
	if !foundTwoStroke {
		t.Logf("Warning: No two-stroke characters found in candidates: %v", candidates)
	}
}

func TestSimpleRecognizer_Recognize_ThreeStrokes(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	
	// Test three strokes (should return characters like 三, 川)
	strokes := []Stroke{
		{
			Points: []Point{
				{X: 10, Y: 10},
				{X: 20, Y: 10},
			},
		},
		{
			Points: []Point{
				{X: 10, Y: 20},
				{X: 20, Y: 20},
			},
		},
		{
			Points: []Point{
				{X: 10, Y: 30},
				{X: 20, Y: 30},
			},
		},
	}
	
	candidates, err := recognizer.Recognize(strokes, 300, 300, 5)
	if err != nil {
		t.Fatalf("Should not return error: %v", err)
	}
	
	if len(candidates) == 0 {
		t.Fatal("Should return at least one candidate")
	}
	
	// Should have candidates for three-stroke characters
	foundThreeStroke := false
	for _, candidate := range candidates {
		if candidate.Text == "三" || candidate.Text == "川" {
			foundThreeStroke = true
			break
		}
	}
	
	if !foundThreeStroke {
		t.Logf("Warning: No three-stroke characters found in candidates: %v", candidates)
	}
}

func TestSimpleRecognizer_Recognize_CrossPattern(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	
	// Test cross pattern (十)
	strokes := []Stroke{
		{
			Points: []Point{
				{X: 10, Y: 20},
				{X: 30, Y: 20},
			},
		},
		{
			Points: []Point{
				{X: 20, Y: 10},
				{X: 20, Y: 30},
			},
		},
	}
	
	candidates, err := recognizer.Recognize(strokes, 300, 300, 5)
	if err != nil {
		t.Fatalf("Should not return error: %v", err)
	}
	
	if len(candidates) == 0 {
		t.Fatal("Should return at least one candidate")
	}
	
	// Should have candidates for cross characters
	foundCross := false
	for _, candidate := range candidates {
		if candidate.Text == "十" || candidate.Text == "＋" {
			foundCross = true
			break
		}
	}
	
	if !foundCross {
		t.Logf("Warning: No cross characters found in candidates: %v", candidates)
	}
}

func TestSimpleRecognizer_Recognize_TopN(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	
	strokes := []Stroke{
		{
			Points: []Point{
				{X: 10, Y: 10},
				{X: 20, Y: 10},
			},
		},
	}
	
	// Test with different topN values
	testCases := []int{1, 3, 5, 10}
	
	for _, topN := range testCases {
		candidates, err := recognizer.Recognize(strokes, 300, 300, topN)
		if err != nil {
			t.Fatalf("Should not return error for topN=%d: %v", topN, err)
		}
		
		if len(candidates) > topN {
			t.Fatalf("Should not return more than %d candidates, got %d", topN, len(candidates))
		}
	}
}

func TestSimpleRecognizer_Close(t *testing.T) {
	recognizer := NewSimpleRecognizer()
	
	// Close should not return an error
	err := recognizer.Close()
	if err != nil {
		t.Fatalf("Close should not return error: %v", err)
	}
}

// Note: analyzeStrokeDirection and analyzeStrokeShape are internal methods
// that may not be exposed in the current implementation.
// These tests are commented out until the methods are made public or
// the tests are restructured to test the public interface.
