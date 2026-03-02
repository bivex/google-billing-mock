package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bivex/google-billing-mock/internal/application/dto"
	"github.com/bivex/google-billing-mock/internal/domain/repository"
	"github.com/bivex/google-billing-mock/internal/infrastructure/mock"
)

// writeJSON serialises v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a Google-style error response.
func writeError(w http.ResponseWriter, code int, message, status string) {
	writeJSON(w, code, dto.ErrorResponse{
		Error: dto.ErrorDetail{Code: code, Message: message, Status: status},
	})
}

// mapError maps domain / infrastructure errors to HTTP status codes.
func mapError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Purchase token not found", "NOT_FOUND")
		return
	}
	var se *mock.ScenarioError
	if errors.As(err, &se) {
		writeError(w, se.Code, se.Message, httpStatusToGoogleStatus(se.Code))
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error(), "INTERNAL")
}

func httpStatusToGoogleStatus(code int) string {
	switch code {
	case 400:
		return "INVALID_ARGUMENT"
	case 401:
		return "UNAUTHENTICATED"
	case 403:
		return "PERMISSION_DENIED"
	case 404:
		return "NOT_FOUND"
	case 409:
		return "ALREADY_EXISTS"
	case 410:
		return "PURCHASE_TOKEN_EXPIRED"
	case 429:
		return "RESOURCE_EXHAUSTED"
	default:
		return "INTERNAL"
	}
}

// decodeJSON decodes the request body into v, writing a 400 on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error(), "INVALID_ARGUMENT")
		return false
	}
	return true
}
