package api

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Code       string `json:"error"`
	Title      string `json:"title"`
	StatusCode int    `json:"-"`
}

var (
	ErrorMethodNotAllowed = Error{
		Code:       "method.not.allowed",
		Title:      "Method not allowed",
		StatusCode: http.StatusMethodNotAllowed,
	}

	ErrorBadRequest = Error{
		Code:       "bad.request",
		Title:      "Bad request",
		StatusCode: http.StatusMethodNotAllowed,
	}
)

func NewAPI() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("/health", HandlerHealth)
	mux.HandleFunc("/run", HandlerParallel)
	mux.HandleFunc("/test", HandlerDemo)

	return mux
}

func RespondOK(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func RespondError(w http.ResponseWriter, err Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)
	_ = json.NewEncoder(w).Encode(err)
}
