package httpserver

import (
	"net/http"
	"strconv"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
)

const (
	defaultUniverseLimit = 40
	maxUniverseLimit     = 200
)

func newBondHandler(service moex.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isin := r.PathValue("isin")
		if !validISIN(isin) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid ISIN"})
			return
		}

		bond, err := service.Bond(r.Context(), isin)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, bond)
	})
}

func newMarketUniverseHandler(service moex.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, ok := parseLimit(r.URL.Query().Get("limit"))
		if !ok {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid limit"})
			return
		}

		universe, err := service.MarketUniverse(r.Context(), limit)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if universe == nil {
			universe = make(moex.MarketUniverse, 0)
		}
		writeJSON(w, http.StatusOK, universe)
	})
}

func validISIN(isin string) bool {
	if len(isin) != 12 {
		return false
	}
	for i := range len(isin) {
		if (isin[i] < 'A' || isin[i] > 'Z') && (isin[i] < '0' || isin[i] > '9') {
			return false
		}
	}
	return true
}

func parseLimit(raw string) (int, bool) {
	if raw == "" {
		return defaultUniverseLimit, true
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maxUniverseLimit {
		return 0, false
	}
	return limit, true
}
