package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

// Board mutation sentinel errors (stable WS/HTTP codes).
var (
	ErrStaleBoard  = errors.New("stale_board")
	ErrOpCancelled = errors.New("op_cancelled")
)

// BoardApplyResult is returned by revision-gated mutators.
type BoardApplyResult struct {
	BoardRev   int64
	StrokeID   int64
	Created    bool
	Deleted    bool
	Cleared    bool
	Idempotent bool
}

type StrokesSnapshot struct {
	BoardRev int64
	Strokes  []Stroke
}

func dbLog(level, format string, args ...interface{}) {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "error":
		if level == "DEBUG" || level == "INFO" || level == "WARN" {
			return
		}
	case "warn":
		if level == "DEBUG" || level == "INFO" {
			return
		}
	case "info":
		if level == "DEBUG" {
			return
		}
	}
	log.Printf(level+" "+format, args...)
}

func (s *Store) ensureBoardStateTx(tx *sql.Tx, userID int64) error {
	_, err := tx.Exec(
		`INSERT OR IGNORE INTO user_board_state(user_id, board_rev) VALUES(?, 0)`,
		userID,
	)
	return err
}

func (s *Store) getBoardRevTx(tx *sql.Tx, userID int64) (int64, error) {
	if err := s.ensureBoardStateTx(tx, userID); err != nil {
		return 0, err
	}
	var rev int64
	err := tx.QueryRow(`SELECT board_rev FROM user_board_state WHERE user_id = ?`, userID).Scan(&rev)
	if err != nil {
		return 0, err
	}
	return rev, nil
}

func (s *Store) bumpBoardRevTx(tx *sql.Tx, userID int64) (int64, error) {
	res, err := tx.Exec(
		`UPDATE user_board_state SET board_rev = board_rev + 1 WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, fmt.Errorf("bumpBoardRev: missing user_board_state userID=%d", userID)
	}
	return s.getBoardRevTx(tx, userID)
}

// GetBoardRev returns the current monotonic board revision for a user (0 if never mutated).
func (s *Store) GetBoardRev(userID int64) (int64, error) {
	tx, err := s.SQL.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	rev, err := s.getBoardRevTx(tx, userID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return rev, nil
}

// ListStrokesWithRev returns strokes plus the current boardRev in one consistent read.
func (s *Store) ListStrokesWithRev(userID int64) (StrokesSnapshot, error) {
	tx, err := s.beginImmediate()
	if err != nil {
		return StrokesSnapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()

	rev, err := s.getBoardRevTx(tx, userID)
	if err != nil {
		return StrokesSnapshot{}, err
	}

	rows, err := tx.Query(
		`SELECT id, color, width, started_at_unix_ms, created_at, COALESCE(op_id, '') FROM strokes WHERE user_id = ? ORDER BY id`,
		userID,
	)
	if err != nil {
		return StrokesSnapshot{}, err
	}
	defer rows.Close()

	var out []Stroke
	for rows.Next() {
		var st Stroke
		st.UserID = userID
		if err := rows.Scan(&st.ID, &st.Color, &st.Width, &st.StartedAtUnixMs, &st.CreatedAt, &st.OpID); err != nil {
			return StrokesSnapshot{}, err
		}
		pr, err := tx.Query(`SELECT x, y FROM stroke_points WHERE stroke_id = ? ORDER BY id`, st.ID)
		if err != nil {
			return StrokesSnapshot{}, err
		}
		for pr.Next() {
			var x, y float64
			if err := pr.Scan(&x, &y); err != nil {
				pr.Close()
				return StrokesSnapshot{}, err
			}
			st.Points = append(st.Points, StrokePoint{X: x, Y: y})
		}
		pr.Close()
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return StrokesSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return StrokesSnapshot{}, err
	}
	return StrokesSnapshot{BoardRev: rev, Strokes: out}, nil
}

func (s *Store) isTombstonedTx(tx *sql.Tx, userID int64, opID string) (bool, error) {
	var n int
	err := tx.QueryRow(
		`SELECT COUNT(1) FROM stroke_op_tombstones WHERE user_id = ? AND op_id = ?`,
		userID, opID,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) insertTombstoneTx(tx *sql.Tx, userID int64, opID, reason string, atRev int64) error {
	_, err := tx.Exec(
		`INSERT OR IGNORE INTO stroke_op_tombstones(user_id, op_id, reason, at_rev) VALUES(?, ?, ?, ?)`,
		userID, opID, reason, atRev,
	)
	return err
}

func (s *Store) boardOpExistsTx(tx *sql.Tx, userID int64, opID string) (exists bool, atRev int64, err error) {
	err = tx.QueryRow(
		`SELECT at_rev FROM board_ops WHERE user_id = ? AND op_id = ?`,
		userID, opID,
	).Scan(&atRev)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, atRev, nil
}

func (s *Store) insertBoardOpTx(tx *sql.Tx, userID int64, opID, kind string, atRev int64) error {
	_, err := tx.Exec(
		`INSERT INTO board_ops(user_id, op_id, kind, at_rev) VALUES(?, ?, ?, ?)`,
		userID, opID, kind, atRev,
	)
	return err
}

func (s *Store) strokeIDByOpTx(tx *sql.Tx, userID int64, opID string) (int64, bool, error) {
	var id int64
	err := tx.QueryRow(
		`SELECT id FROM strokes WHERE user_id = ? AND op_id = ?`,
		userID, opID,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// ApplyStrokeCreate inserts a stroke under strict baseRev equality.
// Active (user_id, op_id) hits return the existing stroke without bumping.
// Tombstoned create opIds return ErrOpCancelled.
func (s *Store) ApplyStrokeCreate(userID, baseRev int64, opID, color string, width int, startedAtUnixMs int64, points []StrokePoint) (BoardApplyResult, error) {
	dbLog("DEBUG", "[db.ApplyStrokeCreate] enter userID=%d baseRev=%d opId=%s", userID, baseRev, opID)
	tx, err := s.beginImmediate()
	if err != nil {
		return BoardApplyResult{}, err
	}
	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	cur, err := s.getBoardRevTx(tx, userID)
	if err != nil {
		return BoardApplyResult{}, err
	}

	tombstoned, err := s.isTombstonedTx(tx, userID, opID)
	if err != nil {
		return BoardApplyResult{}, err
	}
	if tombstoned {
		dbLog("WARN", "[db.ApplyStrokeCreate] cancelled userID=%d opId=%s boardRev=%d", userID, opID, cur)
		return BoardApplyResult{BoardRev: cur}, ErrOpCancelled
	}

	if existingID, ok, lookupErr := s.strokeIDByOpTx(tx, userID, opID); lookupErr != nil {
		return BoardApplyResult{}, lookupErr
	} else if ok {
		dbLog("INFO", "[db.ApplyStrokeCreate] idempotent hit userID=%d opId=%s strokeId=%d boardRev=%d", userID, opID, existingID, cur)
		if err := tx.Commit(); err != nil {
			return BoardApplyResult{}, err
		}
		committed = true
		return BoardApplyResult{BoardRev: cur, StrokeID: existingID, Idempotent: true}, nil
	}

	if baseRev != cur {
		dbLog("WARN", "[db.ApplyStrokeCreate] stale userID=%d baseRev=%d boardRev=%d opId=%s", userID, baseRev, cur, opID)
		return BoardApplyResult{BoardRev: cur}, ErrStaleBoard
	}

	res, err := tx.Exec(
		`INSERT INTO strokes(user_id, color, width, started_at_unix_ms, op_id) VALUES(?, ?, ?, ?, ?)`,
		userID, color, width, startedAtUnixMs, opID,
	)
	if err != nil {
		return BoardApplyResult{}, err
	}
	strokeID, err := res.LastInsertId()
	if err != nil {
		return BoardApplyResult{}, err
	}
	if len(points) > 0 {
		stmt, prepErr := tx.Prepare(`INSERT INTO stroke_points(stroke_id, x, y) VALUES(?, ?, ?)`)
		if prepErr != nil {
			return BoardApplyResult{}, prepErr
		}
		for _, p := range points {
			if _, err = stmt.Exec(strokeID, p.X, p.Y); err != nil {
				_ = stmt.Close()
				return BoardApplyResult{}, err
			}
		}
		_ = stmt.Close()
	}

	newRev, err := s.bumpBoardRevTx(tx, userID)
	if err != nil {
		return BoardApplyResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return BoardApplyResult{}, err
	}
	committed = true
	dbLog("INFO", "[db.ApplyStrokeCreate] applied userID=%d opId=%s strokeId=%d boardRev=%d→%d", userID, opID, strokeID, cur, newRev)
	return BoardApplyResult{BoardRev: newRev, StrokeID: strokeID, Created: true}, nil
}

// ApplyStrokeDelete deletes by stroke id and/or tombstones/deletes by create opId.
// Exactly one of strokeID (>0) or deleteOpID (non-empty) should be provided; both may be set
// (id preferred for the row delete, deleteOpID always tombstones if no active stroke).
func (s *Store) ApplyStrokeDelete(userID, baseRev int64, opID string, strokeID int64, deleteOpID string) (BoardApplyResult, error) {
	dbLog("DEBUG", "[db.ApplyStrokeDelete] enter userID=%d baseRev=%d opId=%s strokeID=%d deleteOpId=%s",
		userID, baseRev, opID, strokeID, deleteOpID)

	tx, err := s.beginImmediate()
	if err != nil {
		return BoardApplyResult{}, err
	}
	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	cur, err := s.getBoardRevTx(tx, userID)
	if err != nil {
		return BoardApplyResult{}, err
	}

	if exists, atRev, opErr := s.boardOpExistsTx(tx, userID, opID); opErr != nil {
		return BoardApplyResult{}, opErr
	} else if exists {
		dbLog("INFO", "[db.ApplyStrokeDelete] idempotent hit userID=%d opId=%s boardRev=%d", userID, opID, atRev)
		if err := tx.Commit(); err != nil {
			return BoardApplyResult{}, err
		}
		committed = true
		return BoardApplyResult{BoardRev: atRev, Deleted: true, Idempotent: true}, nil
	}

	if baseRev != cur {
		dbLog("WARN", "[db.ApplyStrokeDelete] stale userID=%d baseRev=%d boardRev=%d opId=%s", userID, baseRev, cur, opID)
		return BoardApplyResult{BoardRev: cur}, ErrStaleBoard
	}

	deleted := false
	if strokeID > 0 {
		res, delErr := tx.Exec(`DELETE FROM strokes WHERE id = ? AND user_id = ?`, strokeID, userID)
		if delErr != nil {
			return BoardApplyResult{}, delErr
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			deleted = true
		}
	}

	if deleteOpID != "" {
		if existingID, ok, lookupErr := s.strokeIDByOpTx(tx, userID, deleteOpID); lookupErr != nil {
			return BoardApplyResult{}, lookupErr
		} else if ok {
			if _, delErr := tx.Exec(`DELETE FROM strokes WHERE id = ? AND user_id = ?`, existingID, userID); delErr != nil {
				return BoardApplyResult{}, delErr
			}
			deleted = true
		}
		// Always tombstone so a late create cannot resurrect.
		if err := s.insertTombstoneTx(tx, userID, deleteOpID, "delete", cur+1); err != nil {
			return BoardApplyResult{}, err
		}
	}

	newRev, err := s.bumpBoardRevTx(tx, userID)
	if err != nil {
		return BoardApplyResult{}, err
	}
	if err := s.insertBoardOpTx(tx, userID, opID, "delete", newRev); err != nil {
		return BoardApplyResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return BoardApplyResult{}, err
	}
	committed = true
	dbLog("INFO", "[db.ApplyStrokeDelete] applied userID=%d opId=%s deleted=%v boardRev=%d→%d", userID, opID, deleted, cur, newRev)
	return BoardApplyResult{BoardRev: newRev, Deleted: true}, nil
}

// ApplyClear deletes all strokes, tombstones their create opIds, and bumps boardRev.
func (s *Store) ApplyClear(userID, baseRev int64, opID string) (BoardApplyResult, error) {
	dbLog("DEBUG", "[db.ApplyClear] enter userID=%d baseRev=%d opId=%s", userID, baseRev, opID)

	tx, err := s.beginImmediate()
	if err != nil {
		return BoardApplyResult{}, err
	}
	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	cur, err := s.getBoardRevTx(tx, userID)
	if err != nil {
		return BoardApplyResult{}, err
	}

	if exists, atRev, opErr := s.boardOpExistsTx(tx, userID, opID); opErr != nil {
		return BoardApplyResult{}, opErr
	} else if exists {
		dbLog("INFO", "[db.ApplyClear] idempotent hit userID=%d opId=%s boardRev=%d", userID, opID, atRev)
		if err := tx.Commit(); err != nil {
			return BoardApplyResult{}, err
		}
		committed = true
		return BoardApplyResult{BoardRev: atRev, Cleared: true, Idempotent: true}, nil
	}

	if baseRev != cur {
		dbLog("WARN", "[db.ApplyClear] stale userID=%d baseRev=%d boardRev=%d opId=%s", userID, baseRev, cur, opID)
		return BoardApplyResult{BoardRev: cur}, ErrStaleBoard
	}

	rows, err := tx.Query(`SELECT op_id FROM strokes WHERE user_id = ? AND op_id IS NOT NULL AND op_id != ''`, userID)
	if err != nil {
		return BoardApplyResult{}, err
	}
	var createOpIDs []string
	for rows.Next() {
		var oid string
		if err := rows.Scan(&oid); err != nil {
			rows.Close()
			return BoardApplyResult{}, err
		}
		createOpIDs = append(createOpIDs, oid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return BoardApplyResult{}, err
	}

	if _, err := tx.Exec(`DELETE FROM strokes WHERE user_id = ?`, userID); err != nil {
		return BoardApplyResult{}, err
	}

	newRev, err := s.bumpBoardRevTx(tx, userID)
	if err != nil {
		return BoardApplyResult{}, err
	}
	for _, oid := range createOpIDs {
		if err := s.insertTombstoneTx(tx, userID, oid, "clear", newRev); err != nil {
			return BoardApplyResult{}, err
		}
	}
	if err := s.insertBoardOpTx(tx, userID, opID, "clear", newRev); err != nil {
		return BoardApplyResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return BoardApplyResult{}, err
	}
	committed = true
	dbLog("INFO", "[db.ApplyClear] applied userID=%d opId=%s tombstones=%d boardRev=%d→%d", userID, opID, len(createOpIDs), cur, newRev)
	return BoardApplyResult{BoardRev: newRev, Cleared: true}, nil
}

func (s *Store) beginImmediate() (*sql.Tx, error) {
	tx, err := s.SQL.Begin()
	if err != nil {
		return nil, err
	}
	// Upgrade deferred → reserved lock (IMMEDIATE semantics) via a no-op write.
	// UPDATE … WHERE 0 never mutates rows but forces SQLite to take a write lock now.
	if _, err := tx.Exec(`UPDATE user_board_state SET board_rev = board_rev WHERE 0`); err != nil {
		dbLog("WARN", "[FIX][db.beginImmediate] reserved-lock upgrade failed: %v; continuing with deferred tx", err)
	} else {
		dbLog("DEBUG", "[FIX][db.beginImmediate] reserved lock acquired")
	}
	return tx, nil
}
