package httpserver

import (
	"net/http"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/gofiber/fiber/v3"
)

func newCBRRatesHandler(service cbr.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		snapshot, err := service.Snapshot(c.Context())
		if err != nil {
			return writeServiceError(c, err)
		}
		return writeJSON(c, http.StatusOK, snapshot)
	}
}
