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
	if p.Manifest.ContentVersion != "hiragana5-content-v2" {
		t.Fatalf("contentVersion=%q", p.Manifest.ContentVersion)
	}
	if p.Manifest.SchemaVersion != 2 {
		t.Fatalf("schemaVersion=%d", p.Manifest.SchemaVersion)
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

func TestValidateMissingAudioFile(t *testing.T) {
	p, err := curriculum.LoadPublishedV1()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	bad := "/audio/hiragana5/missing.mp3"
	p.Chars[0].Pronunciation.AudioRef = &bad
	err = curriculum.Validate(p)
	if err == nil || !strings.Contains(err.Error(), "missing file") {
		t.Fatalf("want missing file error, got %v", err)
	}
}

func TestValidateEmptyAudioRejected(t *testing.T) {
	p, err := curriculum.LoadPublishedV1()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	p.Chars[0].Pronunciation.AudioRef = nil
	err = curriculum.Validate(p)
	if err == nil || !strings.Contains(err.Error(), "audioRef") {
		t.Fatalf("want audioRef required error, got %v", err)
	}
}

func TestValidateRejectsKanjiExtensionsOnHiragana(t *testing.T) {
	p, err := curriculum.LoadPublishedV1()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	p.Chars[0].KanjiExtensions = &curriculum.KanjiExtensions{Readings: []string{"あ"}}
	err = curriculum.Validate(p)
	if err == nil || !strings.Contains(err.Error(), "kanjiExtensions") {
		t.Fatalf("want kanjiExtensions error, got %v", err)
	}
}

func TestAudioPackRelPath(t *testing.T) {
	rel, err := curriculum.AudioPackRelPath("/audio/hiragana5/a.mp3")
	if err != nil || rel != "audio/a.mp3" {
		t.Fatalf("rel=%q err=%v", rel, err)
	}
	if _, err := curriculum.AudioPackRelPath("https://evil.example/a.mp3"); err == nil {
		t.Fatal("expected reject non-prefix ref")
	}
}
