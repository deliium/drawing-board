package migrations

import "database/sql"

func up0002LearningDomain(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS characters (
		id TEXT PRIMARY KEY,
		set_id TEXT NOT NULL,
		glyph TEXT NOT NULL,
		romanization TEXT,
		stroke_count INTEGER NOT NULL,
		sort_key INTEGER NOT NULL,
		status TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(set_id, glyph)
	);
	CREATE INDEX IF NOT EXISTS idx_characters_set_sort ON characters(set_id, sort_key);

	CREATE TABLE IF NOT EXISTS lessons (
		id TEXT PRIMARY KEY,
		code TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		set_id TEXT NOT NULL,
		sort_order INTEGER NOT NULL,
		status TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS lesson_characters (
		lesson_id TEXT NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
		character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE RESTRICT,
		position INTEGER NOT NULL,
		PRIMARY KEY (lesson_id, character_id),
		UNIQUE (lesson_id, position)
	);

	CREATE TABLE IF NOT EXISTS practice_attempts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE RESTRICT,
		lesson_id TEXT REFERENCES lessons(id) ON DELETE SET NULL,
		status TEXT NOT NULL,
		client_attempt_id TEXT,
		canvas_width INTEGER,
		canvas_height INTEGER,
		started_at TIMESTAMP NOT NULL,
		submitted_at TIMESTAMP,
		assessed_at TIMESTAMP,
		abandoned_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_attempts_user_started ON practice_attempts(user_id, started_at DESC);
	CREATE INDEX IF NOT EXISTS idx_attempts_user_char ON practice_attempts(user_id, character_id, started_at DESC);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_attempts_user_client ON practice_attempts(user_id, client_attempt_id) WHERE client_attempt_id IS NOT NULL;

	CREATE TABLE IF NOT EXISTS attempt_strokes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		attempt_id INTEGER NOT NULL REFERENCES practice_attempts(id) ON DELETE CASCADE,
		seq INTEGER NOT NULL,
		color TEXT NOT NULL DEFAULT '#000000',
		width INTEGER NOT NULL DEFAULT 2,
		started_at_unix_ms INTEGER NOT NULL DEFAULT 0,
		UNIQUE(attempt_id, seq)
	);

	CREATE TABLE IF NOT EXISTS attempt_stroke_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		attempt_stroke_id INTEGER NOT NULL REFERENCES attempt_strokes(id) ON DELETE CASCADE,
		seq INTEGER NOT NULL,
		x REAL NOT NULL,
		y REAL NOT NULL,
		UNIQUE(attempt_stroke_id, seq)
	);
	CREATE INDEX IF NOT EXISTS idx_attempt_stroke_points_stroke ON attempt_stroke_points(attempt_stroke_id);

	CREATE TABLE IF NOT EXISTS assessment_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		attempt_id INTEGER NOT NULL UNIQUE REFERENCES practice_attempts(id) ON DELETE CASCADE,
		pass INTEGER NOT NULL,
		score REAL NOT NULL,
		score_kind TEXT NOT NULL,
		assessor TEXT NOT NULL,
		set_id TEXT NOT NULL,
		reasons_json TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS assessment_feedback (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		assessment_id INTEGER NOT NULL REFERENCES assessment_results(id) ON DELETE CASCADE,
		rank INTEGER NOT NULL,
		code TEXT NOT NULL,
		message TEXT NOT NULL DEFAULT '',
		UNIQUE(assessment_id, rank)
	);

	CREATE TABLE IF NOT EXISTS user_character_progress (
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		character_id TEXT NOT NULL REFERENCES characters(id) ON DELETE RESTRICT,
		status TEXT NOT NULL,
		attempt_count INTEGER NOT NULL DEFAULT 0,
		pass_count INTEGER NOT NULL DEFAULT 0,
		last_attempt_id INTEGER REFERENCES practice_attempts(id) ON DELETE SET NULL,
		last_passed_at TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, character_id)
	);
	`)
	return err
}

// down0002LearningDomain drops learning tables for test-only teardown. Never run on startup.
func down0002LearningDomain(tx *sql.Tx) error {
	_, err := tx.Exec(`
	DROP TABLE IF EXISTS user_character_progress;
	DROP TABLE IF EXISTS assessment_feedback;
	DROP TABLE IF EXISTS assessment_results;
	DROP TABLE IF EXISTS attempt_stroke_points;
	DROP TABLE IF EXISTS attempt_strokes;
	DROP TABLE IF EXISTS practice_attempts;
	DROP TABLE IF EXISTS lesson_characters;
	DROP TABLE IF EXISTS lessons;
	DROP TABLE IF EXISTS characters;
	`)
	return err
}
