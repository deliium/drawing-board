// Package migrations holds numbered SQLite schema Up steps for the versioned runner.
package migrations

import "database/sql"

// Migration is one forward schema step. Down is optional and for tests only.
type Migration struct {
	Version int
	Name    string
	Up      func(tx *sql.Tx) error
	Down    func(tx *sql.Tx) error // nil in production path; tests may call directly
}

// All returns migrations in ascending version order.
func All() []Migration {
	return []Migration{
		{Version: 1, Name: "baseline_board", Up: up0001BaselineBoard},
		{Version: 2, Name: "learning_domain", Up: up0002LearningDomain, Down: down0002LearningDomain},
		{Version: 3, Name: "curriculum_pedagogy", Up: up0003CurriculumPedagogy},
		{Version: 4, Name: "curriculum_ja_pedagogy", Up: up0004CurriculumJaPedagogy},
	}
}

// LatestVersion is the highest registered migration version.
func LatestVersion() int {
	all := All()
	if len(all) == 0 {
		return 0
	}
	return all[len(all)-1].Version
}
