package db

import (
	"errors"
	"os"
	"testing"
)

func TestOpen(t *testing.T) {
	// Create a temporary database file
	tmpFile := "test.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	if store == nil {
		t.Fatal("Store should not be nil")
	}
}

func TestCreateUser(t *testing.T) {
	tmpFile := "test_create_user.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	// Test creating a new user
	userID, err := store.CreateUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if userID == 0 {
		t.Fatal("User ID should not be zero")
	}

	// Test creating duplicate user
	_, err = store.CreateUser("test@example.com", "password456")
	if err == nil {
		t.Fatal("Should not be able to create duplicate user")
	}
}

func TestUpdateUserPasswordHash(t *testing.T) {
	tmpFile := "test_update_password_hash.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.SQL.Close()

	uid, err := store.CreateUser("hash@example.com", "legacy-hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.UpdateUserPasswordHash(uid, "$2a$12$updatedhashplaceholder........"); err != nil {
		t.Fatalf("update: %v", err)
	}
	u, err := store.GetUserByID(uid)
	if err != nil || u == nil {
		t.Fatalf("get: %v", err)
	}
	if u.PasswordHash != "$2a$12$updatedhashplaceholder........" {
		t.Fatalf("hash not updated: %q", u.PasswordHash)
	}
	if err := store.UpdateUserPasswordHash(99999, "x"); err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestGetUserByEmail(t *testing.T) {
	tmpFile := "test_get_user.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	// Create a user first
	createdUserID, err := store.CreateUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test getting existing user
	user, err := store.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.ID != createdUserID {
		t.Fatalf("Expected user ID %d, got %d", createdUserID, user.ID)
	}

	if user.Email != "test@example.com" {
		t.Fatalf("Expected email 'test@example.com', got '%s'", user.Email)
	}

	// Test getting non-existent user
	user2, err := store.GetUserByEmail("nonexistent@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail should not return error for non-existent user, got: %v", err)
	}
	if user2 != nil {
		t.Fatal("Should return nil for non-existent user")
	}
}

func TestSaveStroke(t *testing.T) {
	tmpFile := "test_save_stroke.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	// Create a user first
	userID, err := store.CreateUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a test stroke
	stroke := Stroke{
		UserID: userID,
		Points: []StrokePoint{
			{X: 10, Y: 20},
			{X: 30, Y: 40},
			{X: 50, Y: 60},
		},
		Color: "#000000",
		Width: 2,
	}

	// Save the stroke
	strokeID, err := store.SaveStroke(userID, stroke.Color, stroke.Width, stroke.StartedAtUnixMs, stroke.Points)
	if err != nil {
		t.Fatalf("Failed to save stroke: %v", err)
	}

	if strokeID == 0 {
		t.Fatal("Saved stroke should have an ID")
	}
}

func TestListStrokesByUser(t *testing.T) {
	tmpFile := "test_list_strokes.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	// Create a user first
	userID, err := store.CreateUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create multiple strokes
	stroke1 := Stroke{
		UserID: userID,
		Points: []StrokePoint{{X: 10, Y: 20}, {X: 30, Y: 40}},
		Color:  "#000000",
		Width:  2,
	}

	stroke2 := Stroke{
		UserID: userID,
		Points: []StrokePoint{{X: 50, Y: 60}, {X: 70, Y: 80}},
		Color:  "#FF0000",
		Width:  3,
	}

	_, err = store.SaveStroke(userID, stroke1.Color, stroke1.Width, stroke1.StartedAtUnixMs, stroke1.Points)
	if err != nil {
		t.Fatalf("Failed to save stroke 1: %v", err)
	}

	_, err = store.SaveStroke(userID, stroke2.Color, stroke2.Width, stroke2.StartedAtUnixMs, stroke2.Points)
	if err != nil {
		t.Fatalf("Failed to save stroke 2: %v", err)
	}

	// List strokes for the user
	strokes, err := store.ListStrokesByUser(userID)
	if err != nil {
		t.Fatalf("Failed to list strokes: %v", err)
	}

	if len(strokes) != 2 {
		t.Fatalf("Expected 2 strokes, got %d", len(strokes))
	}
}

func TestClearStrokes(t *testing.T) {
	tmpFile := "test_clear_strokes.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	// Create a user first
	userID, err := store.CreateUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a stroke
	stroke := Stroke{
		UserID: userID,
		Points: []StrokePoint{{X: 10, Y: 20}, {X: 30, Y: 40}},
		Color:  "#000000",
		Width:  2,
	}

	_, err = store.SaveStroke(userID, stroke.Color, stroke.Width, stroke.StartedAtUnixMs, stroke.Points)
	if err != nil {
		t.Fatalf("Failed to save stroke: %v", err)
	}

	// Verify stroke exists
	strokes, err := store.ListStrokesByUser(userID)
	if err != nil {
		t.Fatalf("Failed to list strokes: %v", err)
	}

	if len(strokes) != 1 {
		t.Fatalf("Expected 1 stroke, got %d", len(strokes))
	}

	// Clear strokes
	err = store.ClearStrokesByUser(userID)
	if err != nil {
		t.Fatalf("Failed to clear strokes: %v", err)
	}

	// Verify strokes are cleared
	strokes, err = store.ListStrokesByUser(userID)
	if err != nil {
		t.Fatalf("Failed to list strokes: %v", err)
	}

	if len(strokes) != 0 {
		t.Fatalf("Expected 0 strokes after clear, got %d", len(strokes))
	}
}

func TestDeleteStroke(t *testing.T) {
	tmpFile := "test_delete_stroke.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	// Create a user first
	userID, err := store.CreateUser("test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create a stroke
	stroke := Stroke{
		UserID: userID,
		Points: []StrokePoint{{X: 10, Y: 20}, {X: 30, Y: 40}},
		Color:  "#000000",
		Width:  2,
	}

	strokeID, err := store.SaveStroke(userID, stroke.Color, stroke.Width, stroke.StartedAtUnixMs, stroke.Points)
	if err != nil {
		t.Fatalf("Failed to save stroke: %v", err)
	}

	// Delete the stroke
	err = store.DeleteStroke(userID, strokeID)
	if err != nil {
		t.Fatalf("Failed to delete stroke: %v", err)
	}

	// Verify stroke is deleted
	strokes, err := store.ListStrokesByUser(userID)
	if err != nil {
		t.Fatalf("Failed to list strokes: %v", err)
	}

	if len(strokes) != 0 {
		t.Fatalf("Expected 0 strokes after delete, got %d", len(strokes))
	}
}

func TestListStrokesByUser_CrossUserIsolation(t *testing.T) {
	tmpFile := "test_stroke_privacy.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer store.SQL.Close()

	userA, err := store.CreateUser("a@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user A: %v", err)
	}
	userB, err := store.CreateUser("b@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to create user B: %v", err)
	}

	_, err = store.SaveStroke(userA, "#000000", 2, 1000, []StrokePoint{{X: 1, Y: 2}, {X: 3, Y: 4}})
	if err != nil {
		t.Fatalf("Failed to save stroke for user A: %v", err)
	}

	strokesB, err := store.ListStrokesByUser(userB)
	if err != nil {
		t.Fatalf("Failed to list strokes for user B: %v", err)
	}
	if len(strokesB) != 0 {
		t.Fatalf("privacy regression: ListStrokesByUser(B) must not return strokes saved under user A, got %d", len(strokesB))
	}

	strokesA, err := store.ListStrokesByUser(userA)
	if err != nil {
		t.Fatalf("Failed to list strokes for user A: %v", err)
	}
	if len(strokesA) != 1 {
		t.Fatalf("Expected 1 stroke for user A, got %d", len(strokesA))
	}
}

func TestSaveStrokeIdempotent(t *testing.T) {
	tmpFile := "test_save_stroke_idempotent.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.SQL.Close()

	userID, err := store.CreateUser("idem@example.com", "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pts := []StrokePoint{{X: 1, Y: 2}, {X: 3, Y: 4}}
	id1, created1, err := store.SaveStrokeIdempotent(userID, "op-aaa-bbb-ccc-ddd-eeeeeeeeeeee", "#111111", 2, 100, pts)
	if err != nil || !created1 || id1 == 0 {
		t.Fatalf("first save: id=%d created=%v err=%v", id1, created1, err)
	}

	id2, created2, err := store.SaveStrokeIdempotent(userID, "op-aaa-bbb-ccc-ddd-eeeeeeeeeeee", "#222222", 5, 200, pts)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if created2 {
		t.Fatal("second save should be idempotent hit")
	}
	if id2 != id1 {
		t.Fatalf("expected same id %d, got %d", id1, id2)
	}

	strokes, err := store.ListStrokesByUser(userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(strokes) != 1 {
		t.Fatalf("expected 1 stroke, got %d", len(strokes))
	}
	if strokes[0].OpID != "op-aaa-bbb-ccc-ddd-eeeeeeeeeeee" {
		t.Fatalf("opId: %q", strokes[0].OpID)
	}
}

func TestBoardRev_CreateStaleAndIdempotent(t *testing.T) {
	tmpFile := "test_board_rev_create.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.SQL.Close()

	uid, err := store.CreateUser("board@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pts := []StrokePoint{{X: 1, Y: 2}, {X: 3, Y: 4}}
	r1, err := store.ApplyStrokeCreate(uid, 0, "op-create-1", "#111111", 2, 100, pts)
	if err != nil || !r1.Created || r1.BoardRev != 1 {
		t.Fatalf("create1: %+v err=%v", r1, err)
	}

	r2, err := store.ApplyStrokeCreate(uid, 0, "op-create-2", "#222222", 2, 100, pts)
	if !errors.Is(err, ErrStaleBoard) || r2.BoardRev != 1 {
		t.Fatalf("expected stale_board rev=1, got %+v err=%v", r2, err)
	}

	r3, err := store.ApplyStrokeCreate(uid, 1, "op-create-1", "#333333", 5, 200, pts)
	if err != nil || !r3.Idempotent || r3.StrokeID != r1.StrokeID || r3.BoardRev != 1 {
		t.Fatalf("idempotent: %+v err=%v", r3, err)
	}
}

func TestBoardRev_ClearTombstonesCreate(t *testing.T) {
	tmpFile := "test_board_rev_clear.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.SQL.Close()

	uid, err := store.CreateUser("clear@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	pts := []StrokePoint{{X: 1, Y: 2}, {X: 3, Y: 4}}
	r1, err := store.ApplyStrokeCreate(uid, 0, "op-will-clear", "#111111", 2, 100, pts)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	clr, err := store.ApplyClear(uid, r1.BoardRev, "op-clear-1")
	if err != nil || !clr.Cleared || clr.BoardRev != 2 {
		t.Fatalf("clear: %+v err=%v", clr, err)
	}
	snap, err := store.ListStrokesWithRev(uid)
	if err != nil || len(snap.Strokes) != 0 || snap.BoardRev != 2 {
		t.Fatalf("snap: %+v err=%v", snap, err)
	}
	r2, err := store.ApplyStrokeCreate(uid, 2, "op-will-clear", "#111111", 2, 100, pts)
	if !errors.Is(err, ErrOpCancelled) {
		t.Fatalf("expected op_cancelled, got %+v err=%v", r2, err)
	}
}

func TestBoardRev_DeleteByOpIdBeforeCreate(t *testing.T) {
	tmpFile := "test_board_rev_delete_opid.db"
	defer os.Remove(tmpFile)

	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.SQL.Close()

	uid, err := store.CreateUser("delop@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	del, err := store.ApplyStrokeDelete(uid, 0, "op-del-1", 0, "op-pending-create")
	if err != nil || del.BoardRev != 1 {
		t.Fatalf("delete: %+v err=%v", del, err)
	}
	r, err := store.ApplyStrokeCreate(uid, 1, "op-pending-create", "#111111", 2, 100, []StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}})
	if !errors.Is(err, ErrOpCancelled) {
		t.Fatalf("expected cancelled, got %+v err=%v", r, err)
	}
}

func TestBoardRev_IdempotentClear(t *testing.T) {
	tmpFile := "test_board_rev_idem_clear.db"
	defer os.Remove(tmpFile)
	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.SQL.Close()
	uid, err := store.CreateUser("idemclear@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	_, err = store.ApplyStrokeCreate(uid, 0, "c1", "#111111", 2, 1, []StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	r1, err := store.ApplyClear(uid, 1, "clear-same")
	if err != nil || r1.BoardRev != 2 {
		t.Fatalf("clear1: %+v err=%v", r1, err)
	}
	r2, err := store.ApplyClear(uid, 2, "clear-same")
	if err != nil || !r2.Idempotent || r2.BoardRev != 2 {
		t.Fatalf("clear idempotent: %+v err=%v", r2, err)
	}
}

func TestBoardRev_DeleteByIdBumpsOnce(t *testing.T) {
	tmpFile := "test_board_rev_del_id.db"
	defer os.Remove(tmpFile)
	store, err := Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.SQL.Close()
	uid, err := store.CreateUser("delid@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	cr, err := store.ApplyStrokeCreate(uid, 0, "stroke-op", "#111111", 2, 1, []StrokePoint{{X: 1, Y: 1}, {X: 2, Y: 2}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	d1, err := store.ApplyStrokeDelete(uid, cr.BoardRev, "del-op", cr.StrokeID, "")
	if err != nil || d1.BoardRev != 2 {
		t.Fatalf("delete: %+v err=%v", d1, err)
	}
	d2, err := store.ApplyStrokeDelete(uid, 2, "del-op", cr.StrokeID, "")
	if err != nil || !d2.Idempotent || d2.BoardRev != 2 {
		t.Fatalf("delete idempotent: %+v err=%v", d2, err)
	}
}
