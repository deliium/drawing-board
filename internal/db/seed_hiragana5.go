package db

import (
	"encoding/json"
	"fmt"

	"github.com/deliium/drawing-board/internal/curriculum"
	"github.com/deliium/drawing-board/internal/learn"
)

// SeedHiragana5 upserts five characters, one published lesson, and ordered lesson_characters
// from the reviewed content pack. Idempotent on every Open; fails closed if pack is not published.
func SeedHiragana5(s *Store) error {
	pack, err := curriculum.LoadPublishedV1()
	if err != nil {
		dbLog("ERROR", "[db.seed] pack load failed: %v", err)
		return fmt.Errorf("seed hiragana5 pack: %w", err)
	}

	tx, err := s.beginImmediate()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	setID := pack.Manifest.SetID
	contentVer := pack.Manifest.ContentVersion

	for _, c := range pack.Chars {
		pronJSON, err := json.Marshal(c.Pronunciation)
		if err != nil {
			return fmt.Errorf("seed character %s pronunciation: %w", c.ID, err)
		}
		traceRef := curriculum.TraceRef(c.Glyph)
		_, err = tx.Exec(`
			INSERT INTO characters(
				id, set_id, glyph, romanization, stroke_count, sort_key, status,
				description_en, description_ja, pronunciation_json, example_word, example_romanization,
				example_meaning_en, example_meaning_ja,
				content_version, trace_ref
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				set_id=excluded.set_id,
				glyph=excluded.glyph,
				romanization=excluded.romanization,
				stroke_count=excluded.stroke_count,
				sort_key=excluded.sort_key,
				status=excluded.status,
				description_en=excluded.description_en,
				description_ja=excluded.description_ja,
				pronunciation_json=excluded.pronunciation_json,
				example_word=excluded.example_word,
				example_romanization=excluded.example_romanization,
				example_meaning_en=excluded.example_meaning_en,
				example_meaning_ja=excluded.example_meaning_ja,
				content_version=excluded.content_version,
				trace_ref=excluded.trace_ref,
				updated_at=CURRENT_TIMESTAMP
		`, c.ID, setID, c.Glyph, c.Romanization, c.StrokeCount, c.SortKey, c.Status,
			c.Description.En, c.Description.Ja, string(pronJSON), c.Example.Word, c.Example.Romanization,
			c.Example.MeaningEn, c.Example.MeaningJa,
			contentVer, traceRef)
		if err != nil {
			return fmt.Errorf("seed character %s: %w", c.ID, err)
		}
		dbLog("DEBUG", "[db.seed] character id=%s romanization=%s strokeCount=%d", c.ID, c.Romanization, c.StrokeCount)
	}

	lesson := pack.Manifest.Lesson
	_, err = tx.Exec(`
		INSERT INTO lessons(id, code, title, title_ja, set_id, sort_order, status)
		VALUES(?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(id) DO UPDATE SET
			code=excluded.code,
			title=excluded.title,
			title_ja=excluded.title_ja,
			set_id=excluded.set_id,
			sort_order=excluded.sort_order,
			status=excluded.status,
			updated_at=CURRENT_TIMESTAMP
	`, lesson.ID, lesson.Code, lesson.Title, lesson.TitleJa, setID, learn.LessonStatusPublished)
	if err != nil {
		return fmt.Errorf("seed lesson: %w", err)
	}

	for _, c := range pack.Chars {
		_, err := tx.Exec(`
			INSERT INTO lesson_characters(lesson_id, character_id, position)
			VALUES(?, ?, ?)
			ON CONFLICT(lesson_id, character_id) DO UPDATE SET position=excluded.position
		`, lesson.ID, c.ID, c.SortKey)
		if err != nil {
			return fmt.Errorf("seed lesson_character %s: %w", c.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	dbLog("INFO", "[db.seed] set=%s characters=%d lesson=%s contentVersion=%s contentHash=%s",
		setID, len(pack.Chars), lesson.ID, contentVer, pack.Manifest.ContentHash)
	return nil
}

// Hiragana5Glyphs returns the packed glyph list in sort order (for tests / set alignment).
func Hiragana5Glyphs() []string {
	pack, err := curriculum.LoadPublishedV1()
	if err != nil {
		return nil
	}
	return pack.Glyphs()
}

// Hiragana5ContentVersion returns the published pack contentVersion (empty on load error).
func Hiragana5ContentVersion() string {
	pack, err := curriculum.LoadPublishedV1()
	if err != nil {
		return ""
	}
	return pack.Manifest.ContentVersion
}
