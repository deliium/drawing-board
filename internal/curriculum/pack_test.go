package curriculum_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/deliium/drawing-board/internal/curriculum"
)

func TestLoadPublishedV1(t *testing.T) {
	p, err := curriculum.LoadPublishedV1()
	if err != nil {
		t.Fatalf("LoadPublishedV1: %v", err)
	}
	if p.Manifest.ContentVersion != "hiragana5-content-v1" {
		t.Fatalf("contentVersion=%q", p.Manifest.ContentVersion)
	}
	if len(p.Chars) != 5 {
		t.Fatalf("chars=%d", len(p.Chars))
	}
	glyphs := strings.Join(p.Glyphs(), "")
	if glyphs != "あいうえお" {
		t.Fatalf("glyphs=%q", glyphs)
	}
	t.Logf("hash=%s version=%s", p.Manifest.ContentHash, p.Manifest.ContentVersion)
}

func TestContentHashStable(t *testing.T) {
	p, err := curriculum.LoadPublishedV1()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	h1, err := curriculum.ContentHash(p)
	if err != nil {
		t.Fatalf("hash1: %v", err)
	}
	h2, err := curriculum.ContentHash(p)
	if err != nil {
		t.Fatalf("hash2: %v", err)
	}
	if h1 != h2 || h1 != p.Manifest.ContentHash {
		t.Fatalf("hash drift h1=%s h2=%s manifest=%s", h1, h2, p.Manifest.ContentHash)
	}
}

func TestRefuseDraftsPath(t *testing.T) {
	_, err := curriculum.LoadPack(fstest.MapFS{}, "drafts/wip")
	if err == nil || !strings.Contains(err.Error(), "drafts") {
		t.Fatalf("want drafts refusal, got %v", err)
	}
}

func TestRefuseDraftReviewStatus(t *testing.T) {
	p, err := curriculum.LoadPublishedV1()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	p.Manifest.ReviewStatus = curriculum.ReviewStatusDraft
	if err := curriculum.RequirePublished(p); err == nil {
		t.Fatal("expected RequirePublished to fail for draft")
	}
}
