// Package curriculum loads and validates versioned hiragana5 content packs.
// Log prefix: [curriculum.*] — never dump full stroke coordinates.
package curriculum

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	PackIDHiragana5       = "hiragana5"
	SetIDHiragana5        = "hiragana5"
	ReviewStatusDraft     = "draft"
	ReviewStatusReviewed  = "reviewed"
	ReviewStatusPublished = "published"
	ExpectedCharCount     = 5
	CurrentSchemaVersion  = 2
	// AudioRefStaticPrefix is the URL path prefix served from web/public (and pack audio/).
	AudioRefStaticPrefix = "/audio/hiragana5/"
)

// Point is a normalized [0,1] canvas coordinate.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Pronunciation is pedagogy metadata; audioRef points at a static clip when set.
type Pronunciation struct {
	IPA      string  `json:"ipa"`
	JaHint   string  `json:"jaHint"`
	AudioRef *string `json:"audioRef"`
	Pitch    string  `json:"pitch"`
}

// Description holds learner-facing copy by locale.
type Description struct {
	En string `json:"en"`
	Ja string `json:"ja"`
}

// Example is one short vocabulary gloss.
type Example struct {
	Word         string `json:"word"`
	Romanization string `json:"romanization"`
	MeaningEn    string `json:"meaningEn"`
	MeaningJa    string `json:"meaningJa"`
}

// KanjiExampleSentence is a future kanji pack extension point (unused by hiragana5).
type KanjiExampleSentence struct {
	Text         string `json:"text"`
	Reading      string `json:"reading"`
	MeaningEn    string `json:"meaningEn"`
	MeaningJa    string `json:"meaningJa"`
	Romanization string `json:"romanization,omitempty"`
}

// KanjiExtensions reserves readings/meanings/radicals/sentences for a future kanji pack.
// Hiragana5 must leave this absent or empty; Validate rejects non-empty payloads on hira:*.
type KanjiExtensions struct {
	Readings         []string               `json:"readings,omitempty"`
	Meanings         []string               `json:"meanings,omitempty"`
	Radicals         []string               `json:"radicals,omitempty"`
	ExampleSentences []KanjiExampleSentence `json:"exampleSentences,omitempty"`
}

// Empty reports whether no kanji extension fields are populated.
func (k *KanjiExtensions) Empty() bool {
	if k == nil {
		return true
	}
	return len(k.Readings) == 0 && len(k.Meanings) == 0 && len(k.Radicals) == 0 && len(k.ExampleSentences) == 0
}

// Character is one curriculum glyph record.
type Character struct {
	ID              string           `json:"id"`
	Glyph           string           `json:"glyph"`
	Romanization    string           `json:"romanization"`
	SortKey         int              `json:"sortKey"`
	StrokeCount     int              `json:"strokeCount"`
	Status          string           `json:"status"`
	Pronunciation   Pronunciation    `json:"pronunciation"`
	Description     Description      `json:"description"`
	Guidance        Description      `json:"guidance"`
	Example         Example          `json:"example"`
	KanjiExtensions *KanjiExtensions `json:"kanjiExtensions,omitempty"`
}

// LessonMeta is seeded lesson identity/title.
type LessonMeta struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Title   string `json:"title"`
	TitleJa string `json:"titleJa"`
}

// Manifest is pack-level metadata.
type Manifest struct {
	PackID         string     `json:"packId"`
	ContentVersion string     `json:"contentVersion"`
	SetID          string     `json:"setId"`
	CharacterCount int        `json:"characterCount"`
	SchemaVersion  int        `json:"schemaVersion"`
	ReviewStatus   string     `json:"reviewStatus"`
	ContentHash    string     `json:"contentHash"`
	Licenses       []string   `json:"licenses"`
	Lesson         LessonMeta `json:"lesson"`
}

// GlyphStrokes is ordered polylines for one glyph.
type GlyphStrokes struct {
	Glyph   string    `json:"glyph"`
	Strokes [][]Point `json:"strokes"`
}

// StrokeFile wraps the strokes/traces JSON array.
type StrokeFile struct {
	Characters []GlyphStrokes `json:"characters"`
}

// CharactersFile wraps characters.json.
type CharactersFile struct {
	Characters []Character `json:"characters"`
}

// ReviewRecord is the human sign-off for a published pack.
type ReviewRecord struct {
	Reviewer   string          `json:"reviewer"`
	ReviewedAt string          `json:"reviewedAt"`
	Notes      string          `json:"notes"`
	Checklist  map[string]bool `json:"checklist"`
}

// Pack is a fully loaded content version.
type Pack struct {
	Dir      string
	Manifest Manifest
	Chars    []Character
	Strokes  map[string][][]Point // glyph → strokes
	Traces   map[string][][]Point
	Review   ReviewRecord
	fsys     fs.FS // set by Load*; used to resolve audioRef files
}

func logf(level, format string, args ...interface{}) {
	if !shouldLog(level) {
		return
	}
	log.Printf(level+" "+format, args...)
}

func shouldLog(level string) bool {
	want := strings.ToUpper(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if want == "" {
		want = "INFO"
	}
	order := map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3}
	w, ok := order[want]
	if !ok {
		w = 1
	}
	l, ok := order[strings.ToUpper(level)]
	if !ok {
		return true
	}
	return l >= w
}

// LoadPack reads pack files from fsys under versionDir (e.g. "v1").
// Rejects paths that include "drafts".
func LoadPack(fsys fs.FS, versionDir string) (Pack, error) {
	p, err := loadPackRaw(fsys, versionDir)
	if err != nil {
		return Pack{}, err
	}
	if err := Validate(p); err != nil {
		logf("ERROR", "[curriculum.validate] %v", err)
		return Pack{}, err
	}
	audioN := 0
	for _, c := range p.Chars {
		if c.Pronunciation.AudioRef != nil && strings.TrimSpace(*c.Pronunciation.AudioRef) != "" {
			audioN++
		}
	}
	logf("INFO", "[curriculum.load] contentVersion=%s hash=%s reviewStatus=%s audioFiles=%d",
		p.Manifest.ContentVersion, p.Manifest.ContentHash, p.Manifest.ReviewStatus, audioN)
	return p, nil
}

// LoadPackAllowPlaceholder loads and validates everything except contentHash match.
// Used by contentvalidate -hash to refresh manifest.contentHash.
func LoadPackAllowPlaceholder(fsys fs.FS, versionDir string) (Pack, error) {
	p, err := loadPackRaw(fsys, versionDir)
	if err != nil {
		return Pack{}, err
	}
	orig := p.Manifest.ContentHash
	got, err := ContentHash(p)
	if err != nil {
		return Pack{}, err
	}
	p.Manifest.ContentHash = got
	if err := Validate(p); err != nil {
		p.Manifest.ContentHash = orig
		return Pack{}, err
	}
	p.Manifest.ContentHash = orig
	return p, nil
}

func loadPackRaw(fsys fs.FS, versionDir string) (Pack, error) {
	versionDir = path.Clean(versionDir)
	if versionDir == "." || versionDir == "" {
		return Pack{}, fmt.Errorf("curriculum: empty version dir")
	}
	if strings.Contains(versionDir, "drafts") {
		logf("ERROR", "[curriculum.load] refused drafts path=%s", versionDir)
		return Pack{}, fmt.Errorf("curriculum: refused drafts path %q", versionDir)
	}

	logf("DEBUG", "[curriculum.load] path=%s files=manifest,characters,strokes,traces,review", versionDir)

	var p Pack
	p.Dir = versionDir
	p.fsys = fsys

	if err := readJSON(fsys, path.Join(versionDir, "manifest.json"), &p.Manifest); err != nil {
		return Pack{}, err
	}
	var cf CharactersFile
	if err := readJSON(fsys, path.Join(versionDir, "characters.json"), &cf); err != nil {
		return Pack{}, err
	}
	p.Chars = cf.Characters

	var sf, tf StrokeFile
	if err := readJSON(fsys, path.Join(versionDir, "strokes.json"), &sf); err != nil {
		return Pack{}, err
	}
	if err := readJSON(fsys, path.Join(versionDir, "traces.json"), &tf); err != nil {
		return Pack{}, err
	}
	p.Strokes = strokeMap(sf.Characters)
	p.Traces = strokeMap(tf.Characters)

	if err := readJSON(fsys, path.Join(versionDir, "review.json"), &p.Review); err != nil {
		return Pack{}, err
	}
	return p, nil
}

func readJSON(fsys fs.FS, name string, dest interface{}) error {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("curriculum: read %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("curriculum: parse %s: %w", name, err)
	}
	return nil
}

func strokeMap(list []GlyphStrokes) map[string][][]Point {
	out := make(map[string][][]Point, len(list))
	for _, g := range list {
		out[g.Glyph] = g.Strokes
	}
	return out
}

// ContentHash returns a deterministic SHA-256 hex digest of characters + strokes + traces.
func ContentHash(p Pack) (string, error) {
	type point struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	type glyphPath struct {
		Glyph   string    `json:"glyph"`
		Strokes [][]point `json:"strokes"`
	}
	type payload struct {
		Characters []Character `json:"characters"`
		Strokes    []glyphPath `json:"strokes"`
		Traces     []glyphPath `json:"traces"`
	}

	chars := append([]Character(nil), p.Chars...)
	sort.Slice(chars, func(i, j int) bool { return chars[i].SortKey < chars[j].SortKey })

	glyphs := make([]string, 0, len(chars))
	for _, c := range chars {
		glyphs = append(glyphs, c.Glyph)
	}

	toPaths := func(m map[string][][]Point) []glyphPath {
		out := make([]glyphPath, 0, len(glyphs))
		for _, g := range glyphs {
			strokes := m[g]
			gp := glyphPath{Glyph: g, Strokes: make([][]point, len(strokes))}
			for i, s := range strokes {
				gp.Strokes[i] = make([]point, len(s))
				for j, pt := range s {
					gp.Strokes[i][j] = point(pt)
				}
			}
			out = append(out, gp)
		}
		return out
	}

	raw, err := json.Marshal(payload{
		Characters: chars,
		Strokes:    toPaths(p.Strokes),
		Traces:     toPaths(p.Traces),
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// Validate checks pack invariants. Does not require reviewStatus=published
// (call RequirePublished separately for seed/recognize).
func Validate(p Pack) error {
	m := p.Manifest
	if m.PackID != PackIDHiragana5 {
		return fmt.Errorf("manifest.packId: want %q got %q", PackIDHiragana5, m.PackID)
	}
	if m.SetID != SetIDHiragana5 {
		return fmt.Errorf("manifest.setId: want %q got %q", SetIDHiragana5, m.SetID)
	}
	if m.CharacterCount != ExpectedCharCount {
		return fmt.Errorf("manifest.characterCount: want %d got %d", ExpectedCharCount, m.CharacterCount)
	}
	if m.ContentVersion == "" {
		return fmt.Errorf("manifest.contentVersion: empty")
	}
	if m.SchemaVersion < 1 {
		return fmt.Errorf("manifest.schemaVersion: invalid")
	}
	if m.SchemaVersion > CurrentSchemaVersion {
		return fmt.Errorf("manifest.schemaVersion: unsupported %d (max %d)", m.SchemaVersion, CurrentSchemaVersion)
	}
	switch m.ReviewStatus {
	case ReviewStatusDraft, ReviewStatusReviewed, ReviewStatusPublished:
	default:
		return fmt.Errorf("manifest.reviewStatus: invalid %q", m.ReviewStatus)
	}
	if m.Lesson.ID == "" || m.Lesson.Code == "" || m.Lesson.Title == "" {
		return fmt.Errorf("manifest.lesson: missing id/code/title")
	}
	if m.Lesson.TitleJa == "" {
		return fmt.Errorf("manifest.lesson: missing titleJa")
	}

	if len(p.Chars) != ExpectedCharCount {
		return fmt.Errorf("characters: want %d got %d", ExpectedCharCount, len(p.Chars))
	}

	wantGlyphs := []string{"あ", "い", "う", "え", "お"}
	seenGlyph := map[string]bool{}
	seenID := map[string]bool{}
	seenSort := map[int]bool{}

	for i, c := range p.Chars {
		prefix := fmt.Sprintf("characters[%d]", i)
		if c.ID == "" || !strings.HasPrefix(c.ID, "hira:") {
			return fmt.Errorf("%s.id: invalid %q", prefix, c.ID)
		}
		if seenID[c.ID] {
			return fmt.Errorf("%s.id: duplicate %q", prefix, c.ID)
		}
		seenID[c.ID] = true
		if utf8.RuneCountInString(c.Glyph) != 1 {
			return fmt.Errorf("%s.glyph: want 1 rune got %q", prefix, c.Glyph)
		}
		if seenGlyph[c.Glyph] {
			return fmt.Errorf("%s.glyph: duplicate %q", prefix, c.Glyph)
		}
		seenGlyph[c.Glyph] = true
		if c.Romanization == "" || c.Romanization != strings.ToLower(c.Romanization) {
			return fmt.Errorf("%s.romanization: want lowercase Hepburn, got %q", prefix, c.Romanization)
		}
		if c.SortKey < 1 || c.SortKey > ExpectedCharCount {
			return fmt.Errorf("%s.sortKey: out of range %d", prefix, c.SortKey)
		}
		if seenSort[c.SortKey] {
			return fmt.Errorf("%s.sortKey: duplicate %d", prefix, c.SortKey)
		}
		seenSort[c.SortKey] = true
		if c.Status != "active" && c.Status != "retired" {
			return fmt.Errorf("%s.status: invalid %q", prefix, c.Status)
		}
		if c.Description.En == "" {
			return fmt.Errorf("%s.description.en: empty", prefix)
		}
		if c.Description.Ja == "" {
			return fmt.Errorf("%s.description.ja: empty", prefix)
		}
		if m.SchemaVersion >= 2 {
			if strings.TrimSpace(c.Guidance.En) == "" {
				return fmt.Errorf("%s.guidance.en: empty", prefix)
			}
			if strings.TrimSpace(c.Guidance.Ja) == "" {
				return fmt.Errorf("%s.guidance.ja: empty", prefix)
			}
		}
		if c.Example.Word == "" || c.Example.Romanization == "" || c.Example.MeaningEn == "" {
			return fmt.Errorf("%s.example: incomplete", prefix)
		}
		if c.Example.MeaningJa == "" {
			return fmt.Errorf("%s.example.meaningJa: empty", prefix)
		}
		if !c.KanjiExtensions.Empty() {
			return fmt.Errorf("%s.kanjiExtensions: must be empty for hiragana ids (%s)", prefix, c.ID)
		}

		if err := validateAudioRef(p, prefix, c); err != nil {
			return err
		}

		strokes, ok := p.Strokes[c.Glyph]
		if !ok {
			return fmt.Errorf("%s: missing strokes for glyph %q", prefix, c.Glyph)
		}
		traces, ok := p.Traces[c.Glyph]
		if !ok {
			return fmt.Errorf("%s: missing traces for glyph %q", prefix, c.Glyph)
		}
		if c.StrokeCount != len(strokes) {
			return fmt.Errorf("%s.strokeCount=%d len(strokes)=%d", prefix, c.StrokeCount, len(strokes))
		}
		if len(traces) != c.StrokeCount {
			return fmt.Errorf("%s: len(traces)=%d want %d", prefix, len(traces), c.StrokeCount)
		}
		if err := validatePolylines(prefix+".strokes", strokes); err != nil {
			return err
		}
		if err := validatePolylines(prefix+".traces", traces); err != nil {
			return err
		}
		logf("DEBUG", "[curriculum.validate] glyph=%s strokes=%d", c.Glyph, c.StrokeCount)
	}

	if m.SchemaVersion >= 2 {
		audioOK := 0
		for _, c := range p.Chars {
			if c.Pronunciation.AudioRef != nil && strings.TrimSpace(*c.Pronunciation.AudioRef) != "" {
				audioOK++
			}
		}
		if audioOK != ExpectedCharCount {
			return fmt.Errorf("characters: schemaVersion>=2 requires audioRef for all %d glyphs (got %d)", ExpectedCharCount, audioOK)
		}
	}

	for i, g := range wantGlyphs {
		if !seenGlyph[g] {
			return fmt.Errorf("characters: missing required glyph %q", g)
		}
		// Chart order by sortKey: glyph at sortKey i+1 must be wantGlyphs[i]
		for _, c := range p.Chars {
			if c.SortKey == i+1 && c.Glyph != g {
				return fmt.Errorf("characters: sortKey %d want glyph %q got %q", i+1, g, c.Glyph)
			}
		}
	}

	gotHash, err := ContentHash(p)
	if err != nil {
		return fmt.Errorf("contentHash: %w", err)
	}
	if m.ContentHash == "" || m.ContentHash == "PLACEHOLDER_WILL_BE_REPLACED" {
		return fmt.Errorf("manifest.contentHash: missing or placeholder (got computed %s)", gotHash)
	}
	if !strings.EqualFold(m.ContentHash, gotHash) {
		return fmt.Errorf("manifest.contentHash: mismatch want %s got %s", m.ContentHash, gotHash)
	}

	if m.ReviewStatus == ReviewStatusPublished {
		if strings.TrimSpace(p.Review.Reviewer) == "" {
			return fmt.Errorf("review.reviewer: required when published")
		}
		if strings.EqualFold(p.Review.Reviewer, "ai") || strings.Contains(strings.ToLower(p.Review.Reviewer), "gpt") {
			return fmt.Errorf("review.reviewer: must be a human identity, not %q", p.Review.Reviewer)
		}
		if p.Review.ReviewedAt == "" {
			return fmt.Errorf("review.reviewedAt: required when published")
		}
		if len(p.Review.Checklist) == 0 {
			return fmt.Errorf("review.checklist: required when published")
		}
		for k, v := range p.Review.Checklist {
			if !v {
				return fmt.Errorf("review.checklist.%s: must be true when published", k)
			}
		}
	}
	return nil
}

func validatePolylines(field string, strokes [][]Point) error {
	for i, s := range strokes {
		if len(s) == 0 {
			return fmt.Errorf("%s[%d]: empty polyline", field, i)
		}
		for j, pt := range s {
			if pt.X < 0 || pt.X > 1 || pt.Y < 0 || pt.Y > 1 {
				return fmt.Errorf("%s[%d][%d]: coord out of [0,1] (%g,%g)", field, i, j, pt.X, pt.Y)
			}
		}
	}
	return nil
}

// AudioPackRelPath maps a static audioRef URL to a pack-relative path under versionDir/audio/.
// Example: /audio/hiragana5/a.mp3 → audio/a.mp3
func AudioPackRelPath(audioRef string) (string, error) {
	ref := strings.TrimSpace(audioRef)
	if ref == "" {
		return "", fmt.Errorf("empty audioRef")
	}
	if !strings.HasPrefix(ref, AudioRefStaticPrefix) {
		return "", fmt.Errorf("audioRef %q: want prefix %q", ref, AudioRefStaticPrefix)
	}
	base := path.Base(ref)
	if base == "." || base == "/" || base == "" || strings.Contains(base, "..") {
		return "", fmt.Errorf("audioRef %q: invalid basename", ref)
	}
	ext := strings.ToLower(path.Ext(base))
	switch ext {
	case ".mp3", ".ogg", ".webm", ".wav":
	default:
		return "", fmt.Errorf("audioRef %q: unsupported extension %q", ref, ext)
	}
	return path.Join("audio", base), nil
}

func validateAudioRef(p Pack, prefix string, c Character) error {
	refPtr := c.Pronunciation.AudioRef
	if refPtr == nil || strings.TrimSpace(*refPtr) == "" {
		if p.Manifest.SchemaVersion >= 2 {
			return fmt.Errorf("%s.pronunciation.audioRef: required for schemaVersion>=2", prefix)
		}
		logf("DEBUG", "[curriculum.validate] audioRef=null glyph=%s", c.Glyph)
		return nil
	}
	ref := strings.TrimSpace(*refPtr)
	rel, err := AudioPackRelPath(ref)
	if err != nil {
		logf("WARN", "[curriculum.validate] audio invalid ref=%s: %v", ref, err)
		return fmt.Errorf("%s.pronunciation.audioRef: %w", prefix, err)
	}
	if p.fsys == nil {
		return fmt.Errorf("%s.pronunciation.audioRef: pack filesystem unavailable", prefix)
	}
	full := path.Join(p.Dir, rel)
	info, err := fs.Stat(p.fsys, full)
	if err != nil {
		logf("WARN", "[curriculum.validate] audio missing ref=%s path=%s", ref, full)
		return fmt.Errorf("%s.pronunciation.audioRef: missing file %s", prefix, full)
	}
	if info.IsDir() || info.Size() <= 0 {
		logf("WARN", "[curriculum.validate] audio empty ref=%s path=%s", ref, full)
		return fmt.Errorf("%s.pronunciation.audioRef: empty file %s", prefix, full)
	}
	logf("DEBUG", "[curriculum.validate] audioRef=%s ok bytes=%d glyph=%s", ref, info.Size(), c.Glyph)
	return nil
}

// RequirePublished returns an error unless reviewStatus is published.
func RequirePublished(p Pack) error {
	if p.Manifest.ReviewStatus != ReviewStatusPublished {
		return fmt.Errorf("curriculum: pack not published (status=%s)", p.Manifest.ReviewStatus)
	}
	return nil
}

// Glyphs returns glyphs in sortKey order.
func (p Pack) Glyphs() []string {
	chars := append([]Character(nil), p.Chars...)
	sort.Slice(chars, func(i, j int) bool { return chars[i].SortKey < chars[j].SortKey })
	out := make([]string, len(chars))
	for i, c := range chars {
		out[i] = c.Glyph
	}
	return out
}

// TraceRef returns a stable reference string for a glyph's trace entry.
func TraceRef(glyph string) string {
	return "traces.json#" + glyph
}
