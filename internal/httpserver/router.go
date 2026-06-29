package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	_ "github.com/Chutchev/myinvesthelper_moex_gateway/docs"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/logger"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
	swaggofiber "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewRouter(moexService moex.Service, cbrService cbr.Service, log *logger.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: fiberErrorHandler,
	})

	// Add request logging middleware
	app.Use(RequestLogger(log))

	app.Get("/health", healthHandler)
	app.Get("/v1/bonds/:isin", newBondHandler(moexService))
	app.Get("/v1/bonds", newMarketUniverseHandler(moexService))
	app.Get("/v1/cbr/rates", newCBRRatesHandler(cbrService))
	app.Get("/swagger/*", swaggofiber.HandlerDefault)
	return app
}

func fiberErrorHandler(c fiber.Ctx, err error) error {
	status := http.StatusInternalServerError
	message := "internal server error"
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		status = fiberErr.Code
		message = fiberErr.Message
	}
	return writeJSON(c, status, ErrorResponse{Error: message})
}

func writeJSON(c fiber.Ctx, status int, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Status(status).Send(payload)
}

func writeServiceError(c fiber.Ctx, err error) error {
	if errors.Is(err, apperrors.ErrNotImplemented) {
		c.Locals("error_message", "not implemented")
		c.Locals("error_details", fmt.Sprintf("%+v", err))
		return writeJSON(c, http.StatusNotImplemented, ErrorResponse{Error: "not implemented"})
	}
	c.Locals("error_message", "internal server error")
	c.Locals("error_details", fmt.Sprintf("%+v", err))
	return writeJSON(c, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}
