package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
)

type errorResponse struct {
	Error string `json:"error"`
}

func NewRouter(moexService moex.Service, cbrService cbr.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.Handle("GET /v1/bonds/{isin}", newBondHandler(moexService))
	mux.Handle("GET /v1/bonds", newMarketUniverseHandler(moexService))
	mux.Handle("GET /v1/cbr/rates", newCBRRatesHandler(cbrService))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, apperrors.ErrNotImplemented) {
		writeJSON(w, http.StatusNotImplemented, errorResponse{Error: "not implemented"})
		return
	}
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}
