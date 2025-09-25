package db

import (
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
