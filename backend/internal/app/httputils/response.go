package response

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, statusCode int, message string) {
	JSON(w, statusCode, map[string]string{"error": message})
}