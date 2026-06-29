package httpserver

import (
	"net/http"
	"strconv"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
	"github.com/gofiber/fiber/v3"
)

const (
	defaultUniverseLimit = 40
	maxUniverseLimit     = 200
)

// newBondHandler godoc
// @Summary Get a MOEX bond
// @Tags bonds
// @Produce json
// @Param isin path string true "ISIN" minlength(12) maxlength(12)
// @Success 200 {object} moex.Bond
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 501 {object} ErrorResponse
// @Router /v1/bonds/{isin} [get]
func newBondHandler(service moex.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		isin := c.Params("isin")
		if !validISIN(isin) {
			return writeJSON(c, http.StatusBadRequest, ErrorResponse{Error: "invalid ISIN"})
		}

		bond, err := service.Bond(c.Context(), isin)
		if err != nil {
			return writeServiceError(c, err)
		}
		return writeJSON(c, http.StatusOK, bond)
	}
}

// newMarketUniverseHandler godoc
// @Summary List MOEX bonds
// @Tags bonds
// @Produce json
// @Param limit query int false "Maximum number of bonds" default(40) minimum(1) maximum(200)
// @Success 200 {array} moex.Bond
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 501 {object} ErrorResponse
// @Router /v1/bonds [get]
func newMarketUniverseHandler(service moex.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		limit, ok := parseLimit(c.Query("limit"))
		if !ok {
			return writeJSON(c, http.StatusBadRequest, ErrorResponse{Error: "invalid limit"})
		}

		universe, err := service.MarketUniverse(c.Context(), limit)
		if err != nil {
			return writeServiceError(c, err)
		}
		if universe == nil {
			universe = make(moex.MarketUniverse, 0)
		}
		return writeJSON(c, http.StatusOK, universe)
	}
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
