package httpserver

import "github.com/gofiber/fiber/v3"

type HealthResponse struct {
	Status string `json:"status"`
}

// healthHandler godoc
// @Summary Check gateway health
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(c fiber.Ctx) error {
	return writeJSON(c, fiber.StatusOK, HealthResponse{Status: "ok"})
}
