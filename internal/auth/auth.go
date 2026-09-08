package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
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

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

const (
	sessionName       = "sid"
	minPasswordLength = 8
)

func hashPassword(pw string) string {
	s := sha256.Sum256([]byte(pw))
	return hex.EncodeToString(s[:])
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAuthError(w http.ResponseWriter, code int, errCode, message string) {
	writeJSON(w, code, errorBody{Error: errCode, Message: message})
}

// validEmail performs a lightweight format check: local@domain with at least one label.
func validEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return false
	}
	local := email[:at]
	domain := email[at+1:]
	if local == "" || domain == "" {
		return false
	}
	if strings.ContainsAny(local, " \t\r\n") || strings.ContainsAny(domain, " \t\r\n") {
		return false
	}
	if strings.Contains(domain, ".") {
		parts := strings.Split(domain, ".")
		for _, p := range parts {
			if p == "" {
				return false
			}
		}
		return true
	}
	for _, r := range domain {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func normalizeCredentials(c *credentials) {
	c.Email = strings.TrimSpace(strings.ToLower(c.Email))
}

func validateCredentials(c credentials, requirePasswordMin bool) (code string) {
	if c.Email == "" || c.Password == "" {
		return "missing_fields"
	}
	if !validEmail(c.Email) {
		return "invalid_email"
	}
	if requirePasswordMin && len(c.Password) < minPasswordLength {
		return "password_too_short"
	}
	return ""
}

func validationMessage(code string) string {
	switch code {
	case "missing_fields":
		return "Enter email and password."
	case "invalid_email":
		return "Enter a valid email address."
	case "password_too_short":
		return "Password must be at least 8 characters."
	default:
		return "Something went wrong. Try again."
	}
}

func (s *Service) Register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		authLog("DEBUG", "[auth.Register] code=bad_json")
		writeAuthError(w, http.StatusBadRequest, "bad_json", "Something went wrong. Try again.")
		return
	}
	normalizeCredentials(&c)
	if code := validateCredentials(c, true); code != "" {
		authLog("DEBUG", "[auth.Register] code="+code)
		writeAuthError(w, http.StatusBadRequest, code, validationMessage(code))
		return
	}
	if u, err := s.Store.GetUserByEmail(c.Email); err != nil {
		authLog("ERROR", "[auth.Register] lookup: "+err.Error())
		writeAuthError(w, http.StatusInternalServerError, "registration_failed", "Unable to create account. If you already have one, sign in.")
		return
	} else if u != nil {
		authLog("WARN", "[auth.Register] code=registration_failed reason=email_taken")
		writeAuthError(w, http.StatusBadRequest, "registration_failed", "Unable to create account. If you already have one, sign in.")
		return
	}
	uid, err := s.Store.CreateUser(c.Email, hashPassword(c.Password))
	if err != nil {
		if IsUniqueConstraint(err) {
			authLog("WARN", "[auth.Register] code=registration_failed reason=email_taken")
			writeAuthError(w, http.StatusBadRequest, "registration_failed", "Unable to create account. If you already have one, sign in.")
			return
		}
		authLog("ERROR", "[auth.Register] create: "+err.Error())
		writeAuthError(w, http.StatusInternalServerError, "registration_failed", "Unable to create account. If you already have one, sign in.")
		return
	}
	s.startSession(w, r, uid)
	authLog("INFO", "[auth.Register] userID="+strconv.FormatInt(uid, 10))
	writeJSON(w, http.StatusOK, userView{ID: uid, Email: c.Email})
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		authLog("DEBUG", "[auth.Login] code=bad_json")
		writeAuthError(w, http.StatusBadRequest, "bad_json", "Something went wrong. Try again.")
		return
	}
	normalizeCredentials(&c)
	if code := validateCredentials(c, true); code != "" {
		authLog("DEBUG", "[auth.Login] code="+code)
		writeAuthError(w, http.StatusBadRequest, code, validationMessage(code))
		return
	}
	u, err := s.Store.GetUserByEmail(c.Email)
	if err != nil {
		authLog("ERROR", "[auth.Login] lookup: "+err.Error())
		writeAuthError(w, http.StatusInternalServerError, "invalid_credentials", "Email or password is incorrect.")
		return
	}
	if u == nil || u.PasswordHash != hashPassword(c.Password) {
		authLog("DEBUG", "[auth.Login] code=invalid_credentials")
		writeAuthError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
		return
	}
	s.startSession(w, r, u.ID)
	authLog("INFO", "[auth.Login] userID="+strconv.FormatInt(u.ID, 10))
	writeJSON(w, http.StatusOK, userView{ID: u.ID, Email: u.Email})
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.Sessions.Get(r, sessionName)
	sess.Options.MaxAge = -1 // delete cookie
	_ = sess.Save(r, w)
	authLog("DEBUG", "[auth.Logout] ok")
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (s *Service) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.UserIDFromRequest(r)
	if !ok {
		authLog("DEBUG", "[auth.Me] code=unauthorized")
		writeAuthError(w, http.StatusUnauthorized, "unauthorized", "")
		return
	}
	u, err := s.Store.GetUserByID(uid)
	if err != nil {
		authLog("ERROR", "[auth.Me] lookup userID="+strconv.FormatInt(uid, 10)+": "+err.Error())
		writeAuthError(w, http.StatusInternalServerError, "unauthorized", "")
		return
	}
	if u == nil {
		authLog("DEBUG", "[auth.Me] code=unauthorized missing userID="+strconv.FormatInt(uid, 10))
		writeAuthError(w, http.StatusUnauthorized, "unauthorized", "")
		return
	}
	authLog("DEBUG", "[auth.Me] userID="+strconv.FormatInt(uid, 10))
	writeJSON(w, http.StatusOK, userView{ID: u.ID, Email: u.Email})
}

func (s *Service) UserIDFromRequest(r *http.Request) (int64, bool) {
	sess, err := s.Sessions.Get(r, sessionName)
	if err != nil {
		return 0, false
	}
	v, ok := sess.Values["user_id"].(int64)
	if ok {
		return v, true
	}
	if f, ok := sess.Values["user_id"].(float64); ok {
		return int64(f), true
	}
	return 0, false
}

func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.UserIDFromRequest(r); !ok {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", "")
			return
		}
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

// authLog emits structured level-prefixed lines. DEBUG is suppressed when LOG_LEVEL=info|warn|error.
func authLog(level, msg string) {
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
	log.Printf("%s %s", level, msg)
}
