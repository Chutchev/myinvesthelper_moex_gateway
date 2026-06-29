package httpserver

import (
	"net/http"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/gofiber/fiber/v3"
)

// newCBRRatesHandler godoc
// @Summary Get Bank of Russia rate data
// @Tags cbr
// @Produce json
// @Success 200 {object} cbr.RateSnapshot
// @Failure 500 {object} ErrorResponse
// @Failure 501 {object} ErrorResponse
// @Router /v1/cbr/rates [get]
func newCBRRatesHandler(service cbr.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		snapshot, err := service.Snapshot(c.Context())
		if err != nil {
			return writeServiceError(c, err)
		}
		return writeJSON(c, http.StatusOK, snapshot)
	}
}
