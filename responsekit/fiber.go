package responsekit

import (
	"github.com/gofiber/fiber/v2"
)

// FiberOK writes a 200 success response: {"data": data}.
func FiberOK(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusOK).JSON(successEnvelope(data))
}

// FiberCreated writes a 201 response.
func FiberCreated(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(successEnvelope(data))
}

// FiberAccepted writes a 202 response.
func FiberAccepted(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusAccepted).JSON(successEnvelope(data))
}

// FiberNoContent writes a 204 response with no body.
func FiberNoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// FiberError writes an error response. Status is derived from err
// via statusCode; the body uses errorEnvelope.
func FiberError(c *fiber.Ctx, err error) error {
	return c.Status(statusCode(err)).JSON(errorEnvelope(err))
}

// FiberJSON is a passthrough for non-standard status codes / body
// shapes. Callers are responsible for matching the responsekit
// wire format if they want platform consistency.
func FiberJSON(c *fiber.Ctx, status int, body any) error {
	return c.Status(status).JSON(body)
}