package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/deliium/drawing-board/internal/auth"
	"github.com/deliium/drawing-board/internal/db"
	"github.com/deliium/drawing-board/internal/httpapi"
	"github.com/deliium/drawing-board/internal/recognize"
	"github.com/deliium/drawing-board/internal/ws"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

func main() {
	var (
		addr = flag.String("addr", getEnv("ADDR", ":8080"), "http service address")
		staticDir = flag.String("static", getEnv("STATIC_DIR", ""), "directory to serve static files from (optional)")
		dbPath = flag.String("db", getEnv("DB_PATH", "file:data.db?_fk=1"), "sqlite dsn or file path")
		cookieKey = flag.String("cookie", getEnv("COOKIE_KEY", "change-me-please-32-bytes-min"), "cookie auth key")
		onnxModel = flag.String("onnx_model", getEnv("ONNX_MODEL", "./models/handwriting.onnx"), "path to ONNX model")
	)
	flag.Parse()

	store, err := db.Open(*dbPath)
	if err != nil { log.Fatalf("open db: %v", err) }

	sessionStore := sessions.NewCookieStore([]byte(*cookieKey))
	sessionStore.Options = &sessions.Options{ Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode }
	authSvc := &auth.Service{ Store: store, Sessions: sessionStore }
	
	var recognizer recognize.Recognizer
	if *onnxModel != "" {
		onnxRec, err := recognize.NewONNXRecognizer(*onnxModel)
		if err != nil {
			log.Printf("Warning: failed to initialize ONNX recognizer: %v", err)
			log.Printf("Falling back to simple recognizer")
			recognizer = recognize.NewSimpleRecognizer()
		} else {
			recognizer = onnxRec
		}
	} else {
		recognizer = recognize.NewSimpleRecognizer()
	}
	
	api := &httpapi.API{ Auth: authSvc, Store: store, Recognizer: recognizer }
	ws.Init(store, authSvc)

	r := mux.NewRouter()

	// Auth endpoints
	r.HandleFunc("/api/register", authSvc.Register).Methods(http.MethodPost)
	r.HandleFunc("/api/login", authSvc.Login).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", authSvc.Logout).Methods(http.MethodPost)
	r.HandleFunc("/api/me", authSvc.Me).Methods(http.MethodGet)

	// Strokes endpoints
	r.Handle("/api/strokes", authSvc.RequireAuth(http.HandlerFunc(api.ListStrokes))).Methods(http.MethodGet)
	r.Handle("/api/strokes/clear", authSvc.RequireAuth(http.HandlerFunc(api.ClearStrokes))).Methods(http.MethodPost)
	r.Handle("/api/strokes/delete", authSvc.RequireAuth(http.HandlerFunc(api.DeleteStroke))).Methods(http.MethodPost)
	// Recognize
	r.Handle("/api/recognize", authSvc.RequireAuth(http.HandlerFunc(api.Recognize))).Methods(http.MethodPost)

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

	// Compose middlewares: CORS -> Router, then logging wrapper
	handler := withCORS(r)
	logged := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: 200}
		handler.ServeHTTP(rw, req)
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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
