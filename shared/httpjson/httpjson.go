package httpjson

import (
	"encoding/json"
	"net/http"
)

func Write(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, code int, msg string) {
	Write(w, code, map[string]string{"error": msg})
}

func Decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
