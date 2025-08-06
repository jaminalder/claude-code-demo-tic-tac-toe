package main

import (
	"html/template"

	"htmx-go-app/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.SetFuncMap(template.FuncMap{
		"isHXRequest": func(c *gin.Context) bool {
			return c.GetHeader("HX-Request") == "true"
		},
	})

	r.LoadHTMLGlob("templates/**/*")
	r.Static("/static", "./static")

	// Main pages
	r.GET("/", handlers.HomeHandler)
	r.GET("/new-game", handlers.NewGameHandler)
	r.GET("/game/:id", handlers.GamePageHandler)
	r.GET("/game/:id/select-emoji", handlers.EmojiSelectionHandler)
	r.POST("/game/:id/select-emoji", handlers.EmojiSelectionSubmitHandler)
	
	// Game API endpoints
	r.POST("/api/game/:id/move/:row/:col", handlers.GameMoveHandler)
	r.POST("/api/game/:id/reset", handlers.GameResetHandler)
	r.GET("/api/game/:id/events", handlers.GameSSEHandler)

	r.Run(":8080")
}