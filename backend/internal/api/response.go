package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
}

func conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, gin.H{"error": msg})
}

func internalError(c *gin.Context, err error) {
	log.Printf("internal error [%s %s]: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "errore interno del server"})
}
