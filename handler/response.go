package handler

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func success(c *gin.Context, status int, msg string, data interface{}) {
	c.JSON(status, APIResponse{
		Message: msg,
		Data:    data,
	})
}

func created(c *gin.Context, msg string, data interface{}) {
	c.JSON(201, APIResponse{
		Message: msg,
		Data:    data,
	})
}

func badRequest(c *gin.Context, err string) {
	c.JSON(400, APIResponse{Error: err})
}

func notFound(c *gin.Context, err string) {
	c.JSON(404, APIResponse{Error: err})
}

func internalError(c *gin.Context, err string) {
	c.JSON(500, APIResponse{Error: err})
}

func conflict(c *gin.Context, err string) {
	c.JSON(409, APIResponse{Error: err})
}

