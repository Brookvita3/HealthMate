package helpers

import (
	"storage-service/internal/common"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type OKResponse struct {
	Message string `json:"message"`
}

type DataResponse struct {
	Data interface{} `json:"data"`
}

func HandleError(c *gin.Context, err error) {
	var businessErr *common.BusinessError
	if errors.As(err, &businessErr) {
		c.JSON(businessErr.Code, ErrorResponse{Error: businessErr.Message})
	} else {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "an unexpected error occurred"})
	}
}

func RespondData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}
