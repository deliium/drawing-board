package migrations

import "database/sql"

func up0004CurriculumJaPedagogy(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE characters ADD COLUMN description_ja TEXT NOT NULL DEFAULT '';
	ALTER TABLE characters ADD COLUMN example_meaning_ja TEXT NOT NULL DEFAULT '';
	ALTER TABLE lessons ADD COLUMN title_ja TEXT NOT NULL DEFAULT '';
	`)
	return err
}
