package helpers

type ErrorResponse struct {
	Error string `json:"error"`
}

type OKResponse struct {
	Message string `json:"message"`
}
