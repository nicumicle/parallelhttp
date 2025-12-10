package api

import (
	"math/rand"
	"net/http"
	"time"
)

// HandlerDemo Returns random HTTP status codes with a random duration
func HandlerDemo(w http.ResponseWriter, r *http.Request) {
	n := rand.Intn(10_000)
	time.Sleep(time.Duration(n) * time.Millisecond)

	statuses := []int{
		http.StatusOK,
		http.StatusAccepted,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(statuses[rand.Intn(len(statuses)-1)])
	_, _ = w.Write([]byte(`{"text": "hello"}`))
}
