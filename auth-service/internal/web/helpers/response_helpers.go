package helpers

import (
	"auth-service/internal/common"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleError is a helper to return consistent error responses based on BusinessError.
func HandleError(c *gin.Context, err error) {
	var businessErr *common.BusinessError
	if errors.As(err, &businessErr) {
		c.JSON(businessErr.Code, ErrorResponse{Error: businessErr.Message})
	} else {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "an unexpected error occurred"})
	}
}

// RespondOK sends a JSON response with 200 status and a message.
func RespondOK(c *gin.Context, message string) {
	c.JSON(http.StatusOK, OKResponse{Message: message})
}

// RespondCreated sends a JSON response with 201 status and data.
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// RespondData sends a JSON response with 200 status and data.
func RespondData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}
