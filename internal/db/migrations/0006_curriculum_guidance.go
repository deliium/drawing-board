package migrations

import "database/sql"

func up0006CurriculumGuidance(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE characters ADD COLUMN guidance_en TEXT NOT NULL DEFAULT '';
	ALTER TABLE characters ADD COLUMN guidance_ja TEXT NOT NULL DEFAULT '';
	`)
	return err
}
