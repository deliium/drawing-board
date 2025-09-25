package recognize

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/yalue/onnxruntime_go"
)


type ONNXRecognizer struct {
	session *onnxruntime_go.Session[float32]
	modelPath string
	inputName string
	outputName string
	inputShape []int64
}

func NewONNXRecognizer(modelPath string) (*ONNXRecognizer, error) {
	// Check if the model file exists and is valid
	if modelPath == "" {
		return nil, fmt.Errorf("no model path provided")
	}
	
	// For now, we'll use the improved pattern-based recognition
	// In the future, this could load a real ONNX model
	fmt.Printf("ONNX Recognizer initialized with model path: %s\n", modelPath)
	fmt.Printf("Using advanced pattern-based recognition (ONNX model loading not implemented yet)\n")
	
	return &ONNXRecognizer{
		session: nil,
		modelPath: modelPath,
		inputName: "input",
		outputName: "output", 
		inputShape: []int64{1, 1, 28, 28}, // MNIST-like input shape
	}, nil
}

func (r *ONNXRecognizer) Close() error {
	if r.session != nil {
		r.session.Destroy()
	}
	// Skip ONNX cleanup for mock implementation
	return nil
}

// Convert strokes to a normalized image tensor
func (r *ONNXRecognizer) strokesToTensor(strokes []Stroke, width, height int) ([]float32, error) {
	// Create a grayscale image
	img := image.NewGray(image.Rect(0, 0, width, height))
	
	// Draw strokes directly on the image (no scaling needed since frontend and backend use same coordinates)
	for _, stroke := range strokes {
		if len(stroke.Points) < 1 {
			continue
		}
		
		// Draw all individual points first to ensure nothing is missed
		for _, point := range stroke.Points {
			x := int(point.X)
			y := int(point.Y)
			
			// Draw a 3x3 pixel block around each point for thicker lines
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx := x + dx
					ny := y + dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						img.SetGray(nx, ny, color.Gray{Y: 255}) // White stroke
					}
				}
			}
		}
		
		// Also draw line segments between consecutive points for smoother lines
		for i := 0; i < len(stroke.Points)-1; i++ {
			p1 := stroke.Points[i]
			p2 := stroke.Points[i+1]
			
			// Direct coordinates (no scaling)
			x1 := p1.X
			y1 := p1.Y
			x2 := p2.X
			y2 := p2.Y
			
			// Draw thick line with more steps for smoother lines
			dx := x2 - x1
			dy := y2 - y1
			distance := math.Sqrt(dx*dx + dy*dy)
			steps := int(distance) + 1
			
			for j := 0; j <= steps; j++ {
				t := float64(j) / float64(steps)
				x := int(x1 + t*dx)
				y := int(y1 + t*dy)
				
				// Draw a 3x3 pixel block around each point for thicker lines
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx := x + dx
						ny := y + dy
						if nx >= 0 && nx < width && ny >= 0 && ny < height {
							img.SetGray(nx, ny, color.Gray{Y: 255}) // White stroke
						}
					}
				}
			}
		}
	}
	
	// Convert to tensor (normalize to [0,1] and flatten)
	tensor := make([]float32, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gray := img.GrayAt(x, y)
			tensor[y*width+x] = float32(gray.Y) / 255.0
		}
	}
	
	return tensor, nil
}

func (r *ONNXRecognizer) Recognize(strokes []Stroke, width, height int, topN int) ([]Candidate, error) {
	if topN <= 0 {
		topN = 10
	}
	
	if len(strokes) == 0 {
		return []Candidate{}, nil
	}
	
	// Convert strokes to image tensor for analysis
	tensor, err := r.strokesToTensor(strokes, width, height)
	if err != nil {
		return nil, err
	}
	
	// Analyze the image tensor to extract features
	features := r.analyzeTensorFeatures(tensor, width, height)
	
	// Debug logging with visual representation
	fmt.Printf("Recognition analysis for %d strokes:\n", len(strokes))
	fmt.Printf("  Features: horizontal_lines=%.1f, vertical_lines=%.1f, diagonal_lines=%.1f\n", 
		features["horizontal_lines"], features["vertical_lines"], features["diagonal_lines"])
	fmt.Printf("  Patterns: has_cross=%.1f, has_three_horizontal=%.1f, has_two_horizontal=%.1f\n", 
		features["has_cross"], features["has_three_horizontal"], features["has_two_horizontal"])
	fmt.Printf("  Single: has_single_horizontal=%.1f, has_single_vertical=%.1f\n", 
		features["has_single_horizontal"], features["has_single_vertical"])
	fmt.Printf("  Canvas: width=%d, height=%d, density=%.3f, aspect_ratio=%.2f\n", 
		width, height, features["density"], features["aspect_ratio"])
	
	// Visual debug - show the actual image tensor
	fmt.Printf("  Visual representation (showing active pixels):\n")
	fmt.Printf("  Canvas size: %dx%d, Tensor size: %d\n", width, height, len(tensor))
	
	// Show full canvas with better resolution for debugging
	stepY := 1
	stepX := 1
	if height > 40 {
		stepY = height / 40  // Show more rows
	}
	if width > 80 {
		stepX = width / 80   // Show more columns
	}
	
	for y := 0; y < height; y += stepY {
		fmt.Printf("  ")
		for x := 0; x < width; x += stepX {
			// Sample the pixel value
			idx := y*width + x
			if idx < len(tensor) && tensor[idx] > 0.1 {
				fmt.Printf("█")
			} else {
				fmt.Printf(".")
			}
		}
		fmt.Printf("\n")
	}
	
	// Debug: show actual stroke coordinates and pixel coverage
	fmt.Printf("  Stroke coordinates:\n")
	totalPixels := 0
	for i, stroke := range strokes {
		fmt.Printf("    Stroke %d: %d points\n", i, len(stroke.Points))
		if len(stroke.Points) > 0 {
			first := stroke.Points[0]
			last := stroke.Points[len(stroke.Points)-1]
			fmt.Printf("      First: (%.1f, %.1f), Last: (%.1f, %.1f)\n", 
				first.X, first.Y, last.X, last.Y)
		}
	}
	
	// Count actual pixels drawn
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if tensor[y*width+x] > 0.1 {
				totalPixels++
			}
		}
	}
	fmt.Printf("  Total pixels drawn: %d (%.2f%% of canvas)\n", totalPixels, float64(totalPixels)/float64(width*height)*100)
	
	// Generate candidates based on extracted features
	candidates := r.generateCandidatesFromFeatures(features, len(strokes), topN)
	
	fmt.Printf("  Generated %d candidates: ", len(candidates))
	for i, c := range candidates {
		if i > 0 { fmt.Printf(", ") }
		fmt.Printf("%s(%.2f)", c.Text, c.Score)
	}
	fmt.Printf("\n")
	
	return candidates, nil
}

// analyzeTensorFeatures extracts meaningful features from the image tensor
func (r *ONNXRecognizer) analyzeTensorFeatures(tensor []float32, width, height int) map[string]float64 {
	features := make(map[string]float64)
	
	// Calculate basic statistics
	totalPixels := float64(width * height)
	activePixels := 0.0
	centerX, centerY := float64(width)/2, float64(height)/2
	
	// Find bounding box
	minX, minY := width, height
	maxX, maxY := 0, 0
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if tensor[y*width+x] > 0.1 { // Active pixel
				activePixels++
				if x < minX { minX = x }
				if x > maxX { maxX = x }
				if y < minY { minY = y }
				if y > maxY { maxY = y }
			}
		}
	}
	
	// Basic features
	features["density"] = activePixels / totalPixels
	features["aspect_ratio"] = float64(maxX-minX+1) / float64(maxY-minY+1)
	features["center_offset_x"] = math.Abs(float64(minX+maxX)/2 - centerX) / centerX
	features["center_offset_y"] = math.Abs(float64(minY+maxY)/2 - centerY) / centerY
	
	// Advanced line detection
	horizontalLines := r.detectHorizontalLines(tensor, width, height)
	verticalLines := r.detectVerticalLines(tensor, width, height)
	diagonalLines := r.detectDiagonalLines(tensor, width, height)
	
	features["horizontal_lines"] = float64(horizontalLines)
	features["vertical_lines"] = float64(verticalLines)
	features["diagonal_lines"] = float64(diagonalLines)
	
	// Detect specific patterns
	features["has_cross"] = r.detectCross(tensor, width, height)
	features["has_three_horizontal"] = r.detectThreeHorizontal(tensor, width, height)
	features["has_two_horizontal"] = r.detectTwoHorizontal(tensor, width, height)
	features["has_single_horizontal"] = r.detectSingleHorizontal(tensor, width, height)
	features["has_single_vertical"] = r.detectSingleVertical(tensor, width, height)
	
	return features
}

// detectHorizontalLines finds horizontal line segments
func (r *ONNXRecognizer) detectHorizontalLines(tensor []float32, width, height int) int {
	lines := 0
	minLineLength := width / 10 // More lenient minimum line length
	
	// Group consecutive rows that have horizontal lines
	lineRows := make([]bool, height)
	
	for y := 0; y < height; y++ {
		lineLength := 0
		maxLineLength := 0
		for x := 0; x < width; x++ {
			if tensor[y*width+x] > 0.1 {
				lineLength++
			} else {
				if lineLength > maxLineLength {
					maxLineLength = lineLength
				}
				lineLength = 0
			}
		}
		if lineLength > maxLineLength {
			maxLineLength = lineLength
		}
		lineRows[y] = maxLineLength >= minLineLength
	}
	
	// Count continuous groups of horizontal lines
	inLine := false
	for y := 0; y < height; y++ {
		if lineRows[y] && !inLine {
			lines++
			inLine = true
		} else if !lineRows[y] {
			inLine = false
		}
	}
	
	return lines
}

// detectVerticalLines finds vertical line segments
func (r *ONNXRecognizer) detectVerticalLines(tensor []float32, width, height int) int {
	lines := 0
	minLineLength := height / 10 // More lenient minimum line length
	
	// Group consecutive columns that have vertical lines
	lineCols := make([]bool, width)
	
	for x := 0; x < width; x++ {
		lineLength := 0
		maxLineLength := 0
		for y := 0; y < height; y++ {
			if tensor[y*width+x] > 0.1 {
				lineLength++
			} else {
				if lineLength > maxLineLength {
					maxLineLength = lineLength
				}
				lineLength = 0
			}
		}
		if lineLength > maxLineLength {
			maxLineLength = lineLength
		}
		lineCols[x] = maxLineLength >= minLineLength
	}
	
	// Count continuous groups of vertical lines
	inLine := false
	for x := 0; x < width; x++ {
		if lineCols[x] && !inLine {
			lines++
			inLine = true
		} else if !lineCols[x] {
			inLine = false
		}
	}
	
	return lines
}

// detectDiagonalLines finds diagonal line segments
func (r *ONNXRecognizer) detectDiagonalLines(tensor []float32, width, height int) int {
	lines := 0
	minLineLength := int(math.Sqrt(float64(width*width + height*height))) / 8
	
	// Check diagonal directions
	directions := [][]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	
	for _, dir := range directions {
		dx, dy := dir[0], dir[1]
		for startY := 0; startY < height; startY++ {
			for startX := 0; startX < width; startX++ {
				lineLength := 0
				x, y := startX, startY
				for x >= 0 && x < width && y >= 0 && y < height {
					if tensor[y*width+x] > 0.1 {
						lineLength++
					} else {
						if lineLength >= minLineLength {
							lines++
						}
						lineLength = 0
					}
					x += dx
					y += dy
				}
				if lineLength >= minLineLength {
					lines++
				}
			}
		}
	}
	return lines
}

// detectCross detects if there's a cross pattern (十)
func (r *ONNXRecognizer) detectCross(tensor []float32, width, height int) float64 {
	// Check for horizontal line - look for a line that spans a significant portion of the width
	bestHorizontalLength := 0
	for y := 0; y < height; y++ {
		lineLength := 0
		maxLineLength := 0
		for x := 0; x < width; x++ {
			if tensor[y*width+x] > 0.1 {
				lineLength++
			} else {
				if lineLength > maxLineLength {
					maxLineLength = lineLength
				}
				lineLength = 0
			}
		}
		if lineLength > maxLineLength {
			maxLineLength = lineLength
		}
		if maxLineLength > bestHorizontalLength {
			bestHorizontalLength = maxLineLength
		}
	}
	
	// Check for vertical line - look for a line that spans a significant portion of the height
	bestVerticalLength := 0
	for x := 0; x < width; x++ {
		lineLength := 0
		maxLineLength := 0
		for y := 0; y < height; y++ {
			if tensor[y*width+x] > 0.1 {
				lineLength++
			} else {
				if lineLength > maxLineLength {
					maxLineLength = lineLength
				}
				lineLength = 0
			}
		}
		if lineLength > maxLineLength {
			maxLineLength = lineLength
		}
		if maxLineLength > bestVerticalLength {
			bestVerticalLength = maxLineLength
		}
	}
	
	// Both lines must be present and significant
	if bestHorizontalLength > width/3 && bestVerticalLength > height/3 {
		return 1.0
	}
	return 0.0
}

// detectThreeHorizontal detects three horizontal lines (三)
func (r *ONNXRecognizer) detectThreeHorizontal(tensor []float32, width, height int) float64 {
	horizontalLines := r.detectHorizontalLines(tensor, width, height)
	if horizontalLines >= 3 {
		return 1.0
	}
	return 0.0
}

// detectTwoHorizontal detects two horizontal lines (二)
func (r *ONNXRecognizer) detectTwoHorizontal(tensor []float32, width, height int) float64 {
	horizontalLines := r.detectHorizontalLines(tensor, width, height)
	if horizontalLines >= 2 {
		return 1.0
	}
	return 0.0
}

// detectSingleHorizontal detects a single horizontal line (一)
func (r *ONNXRecognizer) detectSingleHorizontal(tensor []float32, width, height int) float64 {
	horizontalLines := r.detectHorizontalLines(tensor, width, height)
	verticalLines := r.detectVerticalLines(tensor, width, height)
	
	if horizontalLines >= 1 && verticalLines == 0 {
		return 1.0
	}
	return 0.0
}

// detectSingleVertical detects a single vertical line (丨)
func (r *ONNXRecognizer) detectSingleVertical(tensor []float32, width, height int) float64 {
	horizontalLines := r.detectHorizontalLines(tensor, width, height)
	verticalLines := r.detectVerticalLines(tensor, width, height)
	
	if verticalLines >= 1 && horizontalLines == 0 {
		return 1.0
	}
	return 0.0
}

// generateCandidatesFromFeatures creates recognition candidates based on extracted features
func (r *ONNXRecognizer) generateCandidatesFromFeatures(features map[string]float64, strokeCount int, topN int) []Candidate {
	candidates := []Candidate{}
	
	// Priority-based pattern matching using the new detection functions
	
	// Cross detection (十) - highest priority for 2 strokes
	if strokeCount == 2 && features["has_cross"] > 0.5 {
		candidates = append(candidates,
			Candidate{Text: "十", Score: 0.95}, // cross
			Candidate{Text: "＋", Score: 0.8},  // plus
		)
	}
	
	// Three horizontal lines (三) - highest priority for 3 strokes
	if strokeCount == 3 && features["has_three_horizontal"] > 0.5 {
		candidates = append(candidates,
			Candidate{Text: "三", Score: 0.95}, // three horizontal lines
			Candidate{Text: "ミ", Score: 0.7},  // katakana mi
		)
	}
	
	// Two horizontal lines (二) - high priority for 2 strokes
	if strokeCount == 2 && features["has_two_horizontal"] > 0.5 {
		candidates = append(candidates,
			Candidate{Text: "二", Score: 0.9}, // two horizontal lines
			Candidate{Text: "ニ", Score: 0.7}, // katakana ni
		)
	}
	
	// Single horizontal line (一) - high priority for 1 stroke
	if strokeCount == 1 && features["has_single_horizontal"] > 0.5 {
		candidates = append(candidates,
			Candidate{Text: "一", Score: 0.9}, // horizontal line
			Candidate{Text: "ー", Score: 0.7}, // long vowel mark
		)
	}
	
	// Single vertical line (丨) - high priority for 1 stroke
	if strokeCount == 1 && features["has_single_vertical"] > 0.5 {
		candidates = append(candidates,
			Candidate{Text: "丨", Score: 0.9}, // vertical line
			Candidate{Text: "｜", Score: 0.7}, // vertical bar
		)
	}
	
	// Fallback analysis based on line counts
	if len(candidates) == 0 {
		// Single stroke analysis
		if strokeCount == 1 {
			if features["horizontal_lines"] > 0.5 {
				candidates = append(candidates,
					Candidate{Text: "一", Score: 0.7}, // horizontal line
					Candidate{Text: "ー", Score: 0.5}, // long vowel mark
				)
			} else if features["vertical_lines"] > 0.5 {
				candidates = append(candidates,
					Candidate{Text: "丨", Score: 0.7}, // vertical line
					Candidate{Text: "｜", Score: 0.5}, // vertical bar
				)
			} else if features["density"] < 0.01 {
				candidates = append(candidates,
					Candidate{Text: "丶", Score: 0.8}, // dot
					Candidate{Text: "。", Score: 0.6}, // period
				)
			} else {
				candidates = append(candidates,
					Candidate{Text: "し", Score: 0.6}, // curved
					Candidate{Text: "く", Score: 0.4}, // curved
				)
			}
		}
		
		// Two stroke analysis
		if strokeCount == 2 {
			if features["horizontal_lines"] >= 2 {
				candidates = append(candidates,
					Candidate{Text: "二", Score: 0.7}, // two horizontal lines
					Candidate{Text: "ニ", Score: 0.5}, // katakana ni
				)
			} else if features["horizontal_lines"] >= 1 && features["vertical_lines"] >= 1 {
				candidates = append(candidates,
					Candidate{Text: "十", Score: 0.7}, // cross
					Candidate{Text: "＋", Score: 0.5}, // plus
				)
			} else {
				candidates = append(candidates,
					Candidate{Text: "人", Score: 0.6}, // person
					Candidate{Text: "入", Score: 0.4}, // enter
				)
			}
		}
		
		// Three stroke analysis
		if strokeCount == 3 {
			if features["horizontal_lines"] >= 3 {
				candidates = append(candidates,
					Candidate{Text: "三", Score: 0.7}, // three horizontal lines
					Candidate{Text: "ミ", Score: 0.5}, // katakana mi
				)
			} else if features["horizontal_lines"] >= 1 && features["vertical_lines"] >= 1 {
				candidates = append(candidates,
					Candidate{Text: "大", Score: 0.6}, // big
					Candidate{Text: "太", Score: 0.4}, // fat
				)
			} else {
				candidates = append(candidates,
					Candidate{Text: "小", Score: 0.5}, // small
					Candidate{Text: "川", Score: 0.3}, // river
				)
			}
		}
		
		// Complex characters (4+ strokes)
		if strokeCount >= 4 {
			if features["horizontal_lines"] >= 2 && features["vertical_lines"] >= 2 {
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
	}
	
	// Add complexity-based characters
	if features["density"] > 0.1 {
		candidates = append(candidates,
			Candidate{Text: "書", Score: 0.3}, // write
			Candidate{Text: "字", Score: 0.2}, // character
		)
	}
	
	// If no specific matches, provide generic suggestions
	if len(candidates) == 0 {
		if strokeCount == 1 {
			candidates = append(candidates, Candidate{Text: "一", Score: 0.5})
		} else if strokeCount == 2 {
			candidates = append(candidates, Candidate{Text: "二", Score: 0.5})
		} else if strokeCount == 3 {
			candidates = append(candidates, Candidate{Text: "三", Score: 0.5})
		} else {
			candidates = append(candidates, Candidate{Text: "中", Score: 0.4})
		}
	}
	
	// Limit to topN results
	if len(candidates) > topN {
		candidates = candidates[:topN]
	}
	
	return candidates
}
