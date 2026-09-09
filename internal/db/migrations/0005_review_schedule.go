package migrations

import "database/sql"

func up0005ReviewSchedule(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE user_character_progress ADD COLUMN review_box INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE user_character_progress ADD COLUMN due_at TIMESTAMP;
	ALTER TABLE user_character_progress ADD COLUMN last_reviewed_at TIMESTAMP;
	CREATE INDEX IF NOT EXISTS idx_progress_user_due ON user_character_progress(user_id, due_at);
	`)
	return err
}
