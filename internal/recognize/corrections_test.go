package recognize

import "testing"

func TestSelectFeedback_EmptyOnly(t *testing.T) {
	fb := selectFeedback("あ", false, 0, 3, true, false, CriterionScores{}, nil, nil)
	if len(fb) != 1 || fb[0].Code != CodeEmptyStrokes || fb[0].Message == "" {
		t.Fatalf("got %#v", fb)
	}
}

func TestSelectFeedback_AtMostTwo(t *testing.T) {
	cs := CriterionScores{
		StrokeCount:       0.2,
		StrokeOrder:       0.2,
		StartEndDirection: 0.2,
		RelativePlacement: 0.2,
		Proportions:       0.2,
		Shape:             0.2,
	}
	fb := selectFeedback("い", false, 1, 2, false, true, cs, nil, nil)
	if len(fb) > 2 {
		t.Fatalf("want ≤2, got %d %#v", len(fb), fb)
	}
	if len(fb) == 0 || fb[0].Code != CodeStrokeCountMismatch {
		t.Fatalf("want stroke_count_mismatch first, got %#v", fb)
	}
	for _, item := range fb {
		if item.Message == "" {
			t.Fatalf("empty message for %s", item.Code)
		}
	}
}

func TestSelectFeedback_PassStrongEmpty(t *testing.T) {
	cs := CriterionScores{
		StrokeCount: 1, StrokeOrder: 1, StartEndDirection: 1,
		RelativePlacement: 1, Proportions: 1, Shape: 1,
	}
	fb := selectFeedback("う", true, 2, 2, false, false, cs, nil, nil)
	if len(fb) != 0 {
		t.Fatalf("want no feedback on strong pass, got %#v", fb)
	}
}
