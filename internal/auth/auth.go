package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/deliium/drawing-board/internal/db"
	"github.com/gorilla/sessions"
)

type Service struct {
	Store    *db.Store
	Sessions *sessions.CookieStore
}

func NewService(store *db.Store, sessions *sessions.CookieStore) *Service {
	return &Service{
		Store:    store,
		Sessions: sessions,
	}
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userView struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

const sessionName = "sid"

func hashPassword(pw string) string {
	s := sha256.Sum256([]byte(pw))
	return hex.EncodeToString(s[:])
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Service) Register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil { writeJSON(w, 400, map[string]string{"error":"bad json"}); return }
	c.Email = strings.TrimSpace(strings.ToLower(c.Email))
	if c.Email == "" || c.Password == "" { writeJSON(w, 400, map[string]string{"error":"missing fields"}); return }
	if u, _ := s.Store.GetUserByEmail(c.Email); u != nil { writeJSON(w, 409, map[string]string{"error":"email exists"}); return }
	uid, err := s.Store.CreateUser(c.Email, hashPassword(c.Password))
	if err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	s.startSession(w, r, uid)
	writeJSON(w, 200, userView{ID: uid, Email: c.Email})
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil { writeJSON(w, 400, map[string]string{"error":"bad json"}); return }
	u, err := s.Store.GetUserByEmail(strings.TrimSpace(strings.ToLower(c.Email)))
	if err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	if u == nil || u.PasswordHash != hashPassword(c.Password) { writeJSON(w, 401, map[string]string{"error":"invalid credentials"}); return }
	s.startSession(w, r, u.ID)
	writeJSON(w, 200, userView{ID: u.ID, Email: u.Email})
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.Sessions.Get(r, sessionName)
	sess.Options.MaxAge = -1 // delete cookie
	_ = sess.Save(r, w)
	writeJSON(w, 200, map[string]string{"ok":"true"})
}

func (s *Service) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.UserIDFromRequest(r)
	if !ok { writeJSON(w, 401, map[string]string{"error":"unauthorized"}); return }
	u, err := s.Store.GetUserByID(uid)
	if err != nil { writeJSON(w, 500, map[string]string{"error":err.Error()}); return }
	if u == nil { writeJSON(w, 401, map[string]string{"error":"unauthorized"}); return }
	writeJSON(w, 200, userView{ID: u.ID, Email: u.Email})
}

func (s *Service) UserIDFromRequest(r *http.Request) (int64, bool) {
	sess, err := s.Sessions.Get(r, sessionName)
	if err != nil { return 0, false }
	v, ok := sess.Values["user_id"].(int64)
	if ok { return v, true }
	if f, ok := sess.Values["user_id"].(float64); ok { return int64(f), true }
	return 0, false
}

func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.UserIDFromRequest(r); !ok { writeJSON(w, 401, map[string]string{"error":"unauthorized"}); return }
		next.ServeHTTP(w, r)
	})
}

func (s *Service) startSession(w http.ResponseWriter, r *http.Request, userID int64) {
	sess, _ := s.Sessions.Get(r, sessionName)
	sess.Values["user_id"] = userID
	sess.Options.Path = "/"
	sess.Options.HttpOnly = true
	sess.Options.SameSite = http.SameSiteLaxMode
	_ = sess.Save(r, w)
}

func IsUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE")
}

var ErrUnauthorized = errors.New("unauthorized")
