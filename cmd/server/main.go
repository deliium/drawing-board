package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/httpapi"
	"github.com/deliium/drawing-board/internal/limits"
	"github.com/deliium/drawing-board/internal/recognize"
	"github.com/deliium/drawing-board/internal/security"
	"github.com/deliium/drawing-board/internal/ws"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

func main() {
	var (
		addr      = flag.String("addr", getEnv("ADDR", ":8080"), "http service address")
		staticDir = flag.String("static", getEnv("STATIC_DIR", ""), "directory to serve static files from (optional)")
		dbPath    = flag.String("db", getEnv("DB_PATH", "data.db"), "sqlite dsn or file path")
		cookieKey = flag.String("cookie", getEnv("COOKIE_KEY", auth.CookieKeyDefaultSentinel), "cookie auth key")
	)
	flag.Parse()

	if v := os.Getenv("ONNX_MODEL"); v != "" {
		log.Printf("WARN [main] ONNX_MODEL is set but unsupported/removed; ignoring value (use hiragana5 target comparison)")
	}

	appEnv := os.Getenv("APP_ENV")
	secureCookies := auth.ProductionSecureMode(appEnv, os.Getenv("COOKIE_SECURE"))
	log.Printf("INFO [main] cookie_secure=%t", secureCookies)
	if err := auth.ValidateCookieKey(*cookieKey, secureCookies); err != nil {
		log.Fatalf("FATAL [main] COOKIE_KEY validation failed: %v", err)
	}
	if !secureCookies && auth.IsWeakCookieKey(*cookieKey) {
		log.Printf("WARN [main] weak COOKIE_KEY in non-production mode (empty, short, or default sentinel)")
	}

	allowedOrigins, err := security.ResolveAllowedOrigins(appEnv, os.Getenv("ALLOWED_ORIGINS"))
	if err != nil {
		log.Fatalf("FATAL [main] ALLOWED_ORIGINS validation failed: %v", err)
	}
	mode := "development"
	if security.IsProduction(appEnv) {
		mode = "production"
	}
	log.Printf("INFO [main] origin_policy mode=%s count=%d origins=%s", mode, len(allowedOrigins), strings.Join(allowedOrigins, ","))

	store, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	schemaVer, err := store.SchemaVersion()
	if err != nil {
		log.Fatalf("FATAL [main] schema_version: %v", err)
	}
	log.Printf("INFO [main] schema_version=%d learn_seed=hiragana5 contentVersion=%s", schemaVer, db.Hiragana5ContentVersion())
	learnStore := db.NewLearnStore(store)

	sessionStore := sessions.NewCookieStore([]byte(*cookieKey))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies,
	}
	authSvc := auth.NewService(store, sessionStore, secureCookies)

	recognizer, err := recognize.NewTargetCompareRecognizer()
	if err != nil {
		log.Fatalf("FATAL [main] recognizer init failed: %v", err)
	}
	log.Printf("INFO [main] recognizer=target_compare set=%s contentVersion=%s", recognizer.SetID(), recognizer.Version())

	recognize.ConfigureDebug(appEnv, os.Getenv("RECOGNIZE_DEBUG"))

	api := &httpapi.API{
		Auth:             authSvc,
		Store:            store,
		Learn:            learnStore,
		Recognizer:       recognizer,
		Assessor:         recognizer,
		RecognizeLimiter: limits.NewLimiter(limits.RecognizeRatePerMin, limits.RecognizeBurst),
	}
	ws.Init(store, authSvc, allowedOrigins)

	r := mux.NewRouter()

	// Auth endpoints
	r.HandleFunc("/api/register", authSvc.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authSvc.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", authSvc.Logout).Methods(http.MethodPost)
	r.HandleFunc("/api/me", authSvc.Me).Methods(http.MethodGet)
	r.HandleFunc("/api/csrf", security.IssueCSRFHandler(secureCookies)).Methods(http.MethodGet)

	// Strokes endpoints
	r.Handle("/api/strokes", authSvc.RequireAuth(http.HandlerFunc(api.ListStrokes))).Methods(http.MethodGet)
	r.Handle("/api/strokes/clear", authSvc.RequireAuth(http.HandlerFunc(api.ClearStrokes))).Methods(http.MethodPost)
	r.Handle("/api/strokes/delete", authSvc.RequireAuth(http.HandlerFunc(api.DeleteStroke))).Methods(http.MethodPost)
	// Recognize (free-board heuristic only)
	r.Handle("/api/recognize", authSvc.RequireAuth(http.HandlerFunc(api.Recognize))).Methods(http.MethodPost)

	// Practice attempts
	r.Handle("/api/attempts", authSvc.RequireAuth(http.HandlerFunc(api.CreateAttempt))).Methods(http.MethodPost)
	r.Handle("/api/attempts/{id}", authSvc.RequireAuth(http.HandlerFunc(api.GetAttempt))).Methods(http.MethodGet)
	r.Handle("/api/attempts/{id}/submit", authSvc.RequireAuth(http.HandlerFunc(api.SubmitAttempt))).Methods(http.MethodPost)
	r.Handle("/api/attempts/{id}/assess", authSvc.RequireAuth(http.HandlerFunc(api.AssessAttempt))).Methods(http.MethodPost)
	r.Handle("/api/attempts/{id}/assessment", authSvc.RequireAuth(http.HandlerFunc(api.GetAttemptAssessment))).Methods(http.MethodGet)
	r.Handle("/api/attempts/{id}/abandon", authSvc.RequireAuth(http.HandlerFunc(api.AbandonAttempt))).Methods(http.MethodPost)

	// WebSocket endpoint (auth required)
	r.Handle("/ws", authSvc.RequireAuth(http.HandlerFunc(handleWebSocket)))

	// Health check
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods(http.MethodGet)

	// Optionally serve static files (built frontend)
	if *staticDir != "" {
		fs := http.FileServer(http.Dir(*staticDir))
		r.PathPrefix("/").Handler(fs)
	}

	// Middleware (outer → inner): logging → CORS → CSRF → mux
	secured := security.CORS(allowedOrigins, security.CSRF(secureCookies, r))
	logged := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: 200}
		secured.ServeHTTP(rw, req)
		log.Printf("%s %s %d %v", req.Method, req.URL.Path, rw.status, time.Since(start))
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           logged,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

// Implement http.Hijacker passthrough so WebSocket upgrades work through the wrapper
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("hijack not supported")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
