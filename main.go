package main

import (
	"html/template"

	"htmx-go-app/handlers"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/multitemplate"
)

func createMyRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	
	// Define function map
	funcMap := template.FuncMap{
		"isHXRequest": func(c *gin.Context) bool {
			return c.GetHeader("HX-Request") == "true"
		},
	}
	
	// Add templates with base template inheritance
	r.AddFromFilesFuncs("home.html", funcMap, "templates/layouts/base.html", "templates/pages/home.html")
	r.AddFromFilesFuncs("game.html", funcMap, "templates/layouts/base.html", "templates/pages/game.html")
	r.AddFromFilesFuncs("emoji-selection.html", funcMap, "templates/layouts/base.html", "templates/pages/emoji-selection.html")
	r.AddFromFilesFuncs("game-full.html", funcMap, "templates/layouts/base.html", "templates/pages/game-full.html")
	r.AddFromFilesFuncs("404.html", funcMap, "templates/layouts/base.html", "templates/pages/404.html")
	
	return r
}

func main() {
	r := gin.Default()

	r.HTMLRender = createMyRender()
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