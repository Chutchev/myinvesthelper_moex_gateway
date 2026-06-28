package httpserver

import (
	"net/http"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
)

func newCBRRatesHandler(service cbr.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := service.Snapshot(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
	})
}
