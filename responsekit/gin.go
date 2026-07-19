package responsekit

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GinOK writes a 200 success response: {"data": data}.
func GinOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, successEnvelope(data))
}

// GinCreated writes a 201 response.
func GinCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, successEnvelope(data))
}

// GinAccepted writes a 202 response.
func GinAccepted(c *gin.Context, data any) {
	c.JSON(http.StatusAccepted, successEnvelope(data))
}

// GinNoContent writes a 204 response with no body. The explicit
// WriteHeaderNow is required: c.Status only updates the internal
// status field; Gin's writer only flushes on first body write.
func GinNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// GinError writes an error response. Status is derived from err
// via statusCode; the body uses errorEnvelope.
func GinError(c *gin.Context, err error) {
	c.JSON(statusCode(err), errorEnvelope(err))
}

// GinJSON is a passthrough for non-standard status codes / body
// shapes. Callers are responsible for matching the responsekit
// wire format if they want platform consistency.
func GinJSON(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}