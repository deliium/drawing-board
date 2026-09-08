package db

import (
	"fmt"

	"github.com/deliium/drawing-board/internal/recognize"
)

// hiragana5Seed lists stub MVP glyphs (Prompt 10 may replace content; keep ids/set stable).
// Stroke counts match recognize hiragana5 templates.
var hiragana5Seed = []struct {
	ID          string
	Glyph       string
	StrokeCount int
	SortKey     int
}{
	{ID: "hira:あ", Glyph: "あ", StrokeCount: 3, SortKey: 1},
	{ID: "hira:い", Glyph: "い", StrokeCount: 2, SortKey: 2},
	{ID: "hira:う", Glyph: "う", StrokeCount: 2, SortKey: 3},
	{ID: "hira:え", Glyph: "え", StrokeCount: 2, SortKey: 4},
	{ID: "hira:お", Glyph: "お", StrokeCount: 3, SortKey: 5},
}

const (
	seedLessonID    = "lesson:hiragana5"
	seedLessonCode  = "hiragana5"
	seedLessonTitle = "Hiragana あいうえお (stub)"
)

// SeedHiragana5 upserts five characters, one published lesson, and ordered lesson_characters.
// Idempotent on every Open.
func SeedHiragana5(s *Store) error {
	tx, err := s.beginImmediate()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	setID := recognize.SetIDHiragana5
	for _, c := range hiragana5Seed {
		_, err := tx.Exec(`
			INSERT INTO characters(id, set_id, glyph, romanization, stroke_count, sort_key, status)
			VALUES(?, ?, ?, NULL, ?, ?, 'active')
			ON CONFLICT(id) DO UPDATE SET
				set_id=excluded.set_id,
				glyph=excluded.glyph,
				stroke_count=excluded.stroke_count,
				sort_key=excluded.sort_key,
				status='active',
				updated_at=CURRENT_TIMESTAMP
		`, c.ID, setID, c.Glyph, c.StrokeCount, c.SortKey)
		if err != nil {
			return fmt.Errorf("seed character %s: %w", c.ID, err)
		}
		dbLog("DEBUG", "[db.seed] character id=%s glyph=%s", c.ID, c.Glyph)
	}

	_, err = tx.Exec(`
		INSERT INTO lessons(id, code, title, set_id, sort_order, status)
		VALUES(?, ?, ?, ?, 1, 'published')
		ON CONFLICT(id) DO UPDATE SET
			code=excluded.code,
			title=excluded.title,
			set_id=excluded.set_id,
			sort_order=excluded.sort_order,
			status='published',
			updated_at=CURRENT_TIMESTAMP
	`, seedLessonID, seedLessonCode, seedLessonTitle, setID)
	if err != nil {
		return fmt.Errorf("seed lesson: %w", err)
	}

	for _, c := range hiragana5Seed {
		_, err := tx.Exec(`
			INSERT INTO lesson_characters(lesson_id, character_id, position)
			VALUES(?, ?, ?)
			ON CONFLICT(lesson_id, character_id) DO UPDATE SET position=excluded.position
		`, seedLessonID, c.ID, c.SortKey)
		if err != nil {
			return fmt.Errorf("seed lesson_character %s: %w", c.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	dbLog("INFO", "[db.seed] set=%s characters=%d lesson=%s", setID, len(hiragana5Seed), seedLessonID)
	return nil
}

// Hiragana5Glyphs returns the seeded glyph list (for tests / set alignment).
func Hiragana5Glyphs() []string {
	out := make([]string, len(hiragana5Seed))
	for i, c := range hiragana5Seed {
		out[i] = c.Glyph
	}
	return out
}
