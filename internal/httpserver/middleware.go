package httpserver

import (
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/logger"
	"github.com/gofiber/fiber/v3"
)

// RequestLogger creates middleware that logs each HTTP request
func RequestLogger(log *logger.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		requestID := c.Get(fiber.HeaderXRequestID, c.IP())

		// Log request start
		log.Info("request started",
			"method", c.Method(),
			"path", c.Path(),
			"remote_addr", c.IP(),
			"request_id", requestID,
		)

		// Process request
		err := c.Next()

		// Log request completion
		duration := time.Since(start)
		status := c.Response().StatusCode()

		// Get error details from context (set by writeServiceError)
		errorMsg := c.Locals("error_message")
		if errorMsg == nil {
			errorMsg = ""
		}
		errorDetails := c.Locals("error_details")
		if errorDetails == nil {
			errorDetails = ""
		}
		errorStack := c.Locals("error_stack")
		if errorStack == nil {
			errorStack = ""
		}

		// Log errors (status >= 400) at Error level, others at Info
		if status >= 400 {
			// Log to stdout (without stack)
			log.Error("request failed",
				"method", c.Method(),
				"path", c.Path(),
				"status", status,
				"duration", duration.String(),
				"remote_addr", c.IP(),
				"request_id", requestID,
				"error", errorMsg,
				"details", errorDetails,
			)
			
			// Log stack trace to error file separately
			if errorStack != "" {
				log.Error("stack trace",
					"method", c.Method(),
					"path", c.Path(),
					"status", status,
					"stack", errorStack,
				)
			}
		} else {
			log.Info("request completed",
				"method", c.Method(),
				"path", c.Path(),
				"status", status,
				"duration", duration.String(),
				"remote_addr", c.IP(),
				"request_id", requestID,
			)
		}

		return err
	}
}
