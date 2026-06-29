package httpserver

import "github.com/gofiber/fiber/v3"

func healthHandler(c fiber.Ctx) error {
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}
