package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func errorHandlingMiddleware(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("An error occurred: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			}
		}()
		next(c)
	}
}

// InitializeRoutes sets up all the routes for the application
func InitializeRoutes(router *gin.Engine) {
	// Apply the errorHandlingMiddleware to the validateURLsHandler
	router.POST("/validate", errorHandlingMiddleware(validateURLsHandler))

	// Apply the errorHandlingMiddleware to the parseHandler
	router.POST("/parse", errorHandlingMiddleware(parseHandler))

	router.POST("/discover", errorHandlingMiddleware(discoverHandler))

	router.POST("/getreaderview", errorHandlingMiddleware(getReaderViewHandler))
}
