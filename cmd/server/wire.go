package main

import (
	"net/http"

	"github.com/deliium/drawing-board/internal/ws"
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) { ws.Handle(w, r) }
