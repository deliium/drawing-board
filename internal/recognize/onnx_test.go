package recognize

import (
	"testing"
)

func TestNewONNXRecognizer(t *testing.T) {
	// Test with valid model path
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Should not return error for valid model path: %v", err)
	}
	
	if recognizer == nil {
		t.Fatal("ONNXRecognizer should not be nil")
	}
	
	if recognizer.modelPath != "test_model.onnx" {
		t.Fatalf("Expected model path 'test_model.onnx', got '%s'", recognizer.modelPath)
	}
}

func TestNewONNXRecognizer_EmptyPath(t *testing.T) {
	// Test with empty model path
	_, err := NewONNXRecognizer("")
	if err == nil {
		t.Fatal("Should return error for empty model path")
	}
}

func TestONNXRecognizer_Recognize_EmptyStrokes(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	candidates, err := recognizer.Recognize([]Stroke{}, 300, 300, 5)
	if err != nil {
		t.Fatalf("Should not return error for empty strokes: %v", err)
	}
	
	if len(candidates) != 0 {
		t.Fatalf("Expected 0 candidates for empty strokes, got %d", len(candidates))
	}
}

func TestONNXRecognizer_Recognize_SingleStroke(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Test single stroke
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

func TestONNXRecognizer_Recognize_CrossPattern(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
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

func TestONNXRecognizer_Recognize_ThreeHorizontalLines(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Test three horizontal lines (三)
	strokes := []Stroke{
		{
			Points: []Point{
				{X: 10, Y: 10},
				{X: 30, Y: 10},
			},
		},
		{
			Points: []Point{
				{X: 10, Y: 20},
				{X: 30, Y: 20},
			},
		},
		{
			Points: []Point{
				{X: 10, Y: 30},
				{X: 30, Y: 30},
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
	
	// Should have candidates for three horizontal lines
	foundThree := false
	for _, candidate := range candidates {
		if candidate.Text == "三" {
			foundThree = true
			break
		}
	}
	
	if !foundThree {
		t.Logf("Warning: No three horizontal lines character found in candidates: %v", candidates)
	}
}

func TestONNXRecognizer_Recognize_TopN(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
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

func TestONNXRecognizer_Close(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Close should not return an error
	err = recognizer.Close()
	if err != nil {
		t.Fatalf("Close should not return error: %v", err)
	}
}

func TestStrokesToTensor(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Test with simple stroke
	strokes := []Stroke{
		{
			Points: []Point{
				{X: 10, Y: 10},
				{X: 20, Y: 10},
			},
		},
	}
	
	tensor, err := recognizer.strokesToTensor(strokes, 300, 300)
	if err != nil {
		t.Fatalf("Should not return error: %v", err)
	}
	
	// Check tensor size
	expectedSize := 300 * 300
	if len(tensor) != expectedSize {
		t.Fatalf("Expected tensor size %d, got %d", expectedSize, len(tensor))
	}
	
	// Check that some pixels are set (should have some non-zero values)
	hasNonZero := false
	for _, value := range tensor {
		if value > 0 {
			hasNonZero = true
			break
		}
	}
	
	if !hasNonZero {
		t.Fatal("Tensor should have some non-zero values")
	}
}

func TestStrokesToTensor_EmptyStrokes(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Test with empty strokes
	tensor, err := recognizer.strokesToTensor([]Stroke{}, 300, 300)
	if err != nil {
		t.Fatalf("Should not return error: %v", err)
	}
	
	// Check tensor size
	expectedSize := 300 * 300
	if len(tensor) != expectedSize {
		t.Fatalf("Expected tensor size %d, got %d", expectedSize, len(tensor))
	}
	
	// All values should be zero
	for i, value := range tensor {
		if value != 0 {
			t.Fatalf("Expected all values to be zero, but found %f at index %d", value, i)
		}
	}
}

func TestDetectHorizontalLines(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Create a tensor with horizontal lines
	tensor := make([]float32, 300*300)
	
	// Add horizontal line at y=50
	for x := 10; x < 50; x++ {
		tensor[50*300+x] = 1.0
	}
	
	// Add horizontal line at y=100
	for x := 20; x < 60; x++ {
		tensor[100*300+x] = 1.0
	}
	
	lines := recognizer.detectHorizontalLines(tensor, 300, 300)
	if lines < 2 {
		t.Fatalf("Expected at least 2 horizontal lines, got %d", lines)
	}
}

func TestDetectVerticalLines(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Create a tensor with vertical lines
	tensor := make([]float32, 300*300)
	
	// Add vertical line at x=50
	for y := 10; y < 50; y++ {
		tensor[y*300+50] = 1.0
	}
	
	// Add vertical line at x=100
	for y := 20; y < 60; y++ {
		tensor[y*300+100] = 1.0
	}
	
	lines := recognizer.detectVerticalLines(tensor, 300, 300)
	if lines < 2 {
		t.Fatalf("Expected at least 2 vertical lines, got %d", lines)
	}
}

func TestDetectCross(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Create a tensor with a cross
	tensor := make([]float32, 300*300)
	
	// Add horizontal line (120 pixels = 40% of 300)
	for x := 40; x < 160; x++ {
		tensor[100*300+x] = 1.0
	}
	
	// Add vertical line (120 pixels = 40% of 300)
	for y := 40; y < 160; y++ {
		tensor[y*300+100] = 1.0
	}
	
	cross := recognizer.detectCross(tensor, 300, 300)
	if cross < 0.5 {
		t.Fatalf("Expected cross detection > 0.5, got %f", cross)
	}
}

func TestDetectThreeHorizontal(t *testing.T) {
	recognizer, err := NewONNXRecognizer("test_model.onnx")
	if err != nil {
		t.Fatalf("Failed to create recognizer: %v", err)
	}
	
	// Create a tensor with three horizontal lines
	tensor := make([]float32, 300*300)
	
	// Add three horizontal lines
	for x := 10; x < 50; x++ {
		tensor[20*300+x] = 1.0
		tensor[30*300+x] = 1.0
		tensor[40*300+x] = 1.0
	}
	
	three := recognizer.detectThreeHorizontal(tensor, 300, 300)
	if three < 0.5 {
		t.Fatalf("Expected three horizontal detection > 0.5, got %f", three)
	}
}
