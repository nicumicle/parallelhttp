package api

import (
	"net/http"
)

func HandlerHealth(w http.ResponseWriter, r *http.Request) {
	RespondOK(w, []byte(`{"status": "ok"}`))
}
