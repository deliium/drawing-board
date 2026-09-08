package migrations

import "database/sql"

func up0003CurriculumPedagogy(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE characters ADD COLUMN description_en TEXT NOT NULL DEFAULT '';
	ALTER TABLE characters ADD COLUMN pronunciation_json TEXT NOT NULL DEFAULT '{}';
	ALTER TABLE characters ADD COLUMN example_word TEXT NOT NULL DEFAULT '';
	ALTER TABLE characters ADD COLUMN example_romanization TEXT NOT NULL DEFAULT '';
	ALTER TABLE characters ADD COLUMN example_meaning_en TEXT NOT NULL DEFAULT '';
	ALTER TABLE characters ADD COLUMN content_version TEXT NOT NULL DEFAULT '';
	ALTER TABLE characters ADD COLUMN trace_ref TEXT NOT NULL DEFAULT '';
	`)
	return err
}
