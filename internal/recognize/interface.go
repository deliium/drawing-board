package recognize

// Recognizer interface for different recognition implementations
type Recognizer interface {
	Recognize(strokes []Stroke, width, height int, topN int) ([]Candidate, error)
	Close() error
}

// Types for stroke recognition
type Point struct { X float64 `json:"x"`; Y float64 `json:"y"` }
type Stroke struct { Points []Point `json:"points"` }
type Candidate struct { Text string `json:"text"`; Score float64 `json:"score"` }
