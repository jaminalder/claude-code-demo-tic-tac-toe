package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"htmx-go-app/events"
	"htmx-go-app/game"
	"htmx-go-app/models"

	"github.com/gin-gonic/gin"
)



func getPlayerIDFromContext(c *gin.Context) string {
	// Simple approach: use session cookie or generate new ID
	playerID, err := c.Cookie("player_id")
	if err != nil || playerID == "" {
		playerID = game.GeneratePlayerID()
		c.SetCookie("player_id", playerID, 3600*24, "/", "", false, true)
	}
	return playerID
}


func HomeHandler(c *gin.Context) {
	data := gin.H{
		"Title": "Tic-Tac-Toe Game",
	}

	c.HTML(http.StatusOK, "home.html", data)
}

func NewGameHandler(c *gin.Context) {
	newGame := game.CreateGame()
	c.Redirect(http.StatusSeeOther, "/game/"+newGame.ID+"/select-emoji")
}

func GamePageHandler(c *gin.Context) {
	gameID := c.Param("id")
	gameData := game.GetGame(gameID)

	if gameData == nil {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"Title": "Game Not Found",
		})
		return
	}

	// Check if player has selected emoji
	playerID := getPlayerIDFromContext(c)
	player, playerExists := gameData.Players[playerID]

	if !playerExists || player.Emoji == "" {
		// Redirect to emoji selection
		c.Redirect(http.StatusSeeOther, "/game/"+gameID+"/select-emoji")
		return
	}

	// Only allow access when game is ready (2 players)
	if !game.IsGameReady(gameData) {
		// Redirect back to emoji selection (will show waiting state if needed)
		c.Redirect(http.StatusSeeOther, "/game/"+gameID+"/select-emoji")
		return
	}

	// Get player list for display
	var playerEmojis []string
	for _, pID := range gameData.PlayerOrder {
		if p, exists := gameData.Players[pID]; exists {
			playerEmojis = append(playerEmojis, p.Emoji)
		}
	}

	// Get current turn information
	currentTurnPlayerID := game.GetCurrentPlayerID(gameData)
	var currentTurnEmoji string
	if currentTurnPlayerID != "" {
		if currentPlayer, exists := gameData.Players[currentTurnPlayerID]; exists {
			currentTurnEmoji = currentPlayer.Emoji
		}
	}

	// Get winner information
	var winnerEmoji string
	if gameData.Winner != "" {
		if winner, exists := gameData.Players[gameData.Winner]; exists {
			winnerEmoji = winner.Emoji
		}
	}

	data := gin.H{
		"Title":            "Tic-Tac-Toe Game #" + gameID,
		"GameID":           gameID,
		"PlayerEmojis":     playerEmojis,
		"CurrentPlayer":    player,
		"GameStatus":       gameData.Status,
		"CurrentTurnEmoji": currentTurnEmoji,
		"IsPlayersTurn":    game.IsPlayersTurn(gameData, playerID),
		"WinnerEmoji":      winnerEmoji,
		"IsGameActive":     game.IsGameActive(gameData),
		"IsGameFinished":   game.IsGameFinished(gameData),
	}

	c.HTML(http.StatusOK, "game.html", data)
}

func EmojiSelectionHandler(c *gin.Context) {
	gameID := c.Param("id")
	gameData := game.GetGame(gameID)

	if gameData == nil {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"Title": "Game Not Found",
		})
		return
	}

	playerID := getPlayerIDFromContext(c)

	// Check if game is full
	if !game.CanJoinGame(gameData) {
		// Check if this player is already in the game
		if _, exists := gameData.Players[playerID]; !exists {
			c.HTML(http.StatusOK, "game-full.html", gin.H{
				"Title": "Game Full",
			})
			return
		}
	}

	// If player already has emoji selected
	if player, exists := gameData.Players[playerID]; exists && player.Emoji != "" {
		// Check if this is the first player and game is still waiting
		if game.IsFirstPlayer(gameData, playerID) && gameData.Status == models.GameStatusWaiting {
			// Show waiting state
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			host := c.Request.Host
			gameURL := fmt.Sprintf("%s://%s/game/%s", scheme, host, gameID)

			data := gin.H{
				"Title":          "Waiting for Opponent",
				"GameID":         gameID,
				"GameURL":        gameURL,
				"SelectedEmoji":  player.Emoji,
				"IsWaitingState": true,
				"IsFirstPlayer":  true,
			}
			c.HTML(http.StatusOK, "emoji-selection.html", data)
			return
		}

		// If game is ready, redirect to game
		if game.IsGameReady(gameData) {
			c.Redirect(http.StatusSeeOther, "/game/"+gameID)
			return
		}
	}

	// Get available emojis (not taken by other players)
	var availableEmojiList []map[string]interface{}
	for _, emoji := range models.AvailableEmojis {
		available := game.IsEmojiAvailable(gameData, emoji)
		availableEmojiList = append(availableEmojiList, map[string]interface{}{
			"emoji":     emoji,
			"available": available,
		})
	}

	// Determine if this would be the first player
	wouldBeFirst := len(gameData.Players) == 0

	data := gin.H{
		"Title":           "Select Your Emoji",
		"GameID":          gameID,
		"AvailableEmojis": availableEmojiList,
		"IsWaitingState":  false,
		"IsFirstPlayer":   wouldBeFirst,
	}

	c.HTML(http.StatusOK, "emoji-selection.html", data)
}

func EmojiSelectionSubmitHandler(c *gin.Context) {
	gameID := c.Param("id")
	gameData := game.GetGame(gameID)

	if gameData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	playerID := getPlayerIDFromContext(c)
	selectedEmoji := c.PostForm("emoji")

	if selectedEmoji == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No emoji selected"})
		return
	}

	isFirstPlayerJoining := len(gameData.Players) == 0
	err := game.AddPlayerToGame(gameData, playerID, selectedEmoji)
	isGameReadyNow := gameData.Status == models.GameStatusActive

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Broadcast player join event
	events.BroadcastGameEvent(gameID, models.GameEvent{
		Type:   "player_join",
		GameID: gameID,
		Data: map[string]interface{}{
			"playerID": playerID,
			"emoji":    selectedEmoji,
		},
	})

	if isFirstPlayerJoining {
		// First player stays in waiting state (will be shown by EmojiSelectionHandler)
		c.Redirect(http.StatusSeeOther, "/game/"+gameID+"/select-emoji")
	} else if isGameReadyNow {
		// Second player joining - game is active, both players enter
		events.BroadcastGameEvent(gameID, models.GameEvent{
			Type:   "game_ready",
			GameID: gameID,
			Data: map[string]interface{}{
				"status": "active",
			},
		})
		c.Redirect(http.StatusSeeOther, "/game/"+gameID)
	} else {
		// Fallback
		c.Redirect(http.StatusSeeOther, "/game/"+gameID+"/select-emoji")
	}
}


func GameMoveHandler(c *gin.Context) {
	if c.GetHeader("HX-Request") != "true" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HTMX request required"})
		return
	}

	gameID := c.Param("id")
	rowStr := c.Param("row")
	colStr := c.Param("col")

	gameData := game.GetGame(gameID)
	if gameData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Get player ID and check if player exists
	playerID := getPlayerIDFromContext(c)
	player, exists := gameData.Players[playerID]
	if !exists || player.Emoji == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Player not registered"})
		return
	}

	row, err := strconv.Atoi(rowStr)
	if err != nil || row < 0 || row > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid row"})
		return
	}

	col, err := strconv.Atoi(colStr)
	if err != nil || col < 0 || col > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column"})
		return
	}

	// Check if game is finished
	if game.IsGameFinished(gameData) {
		renderGameBoard(c, gameID)
		return
	}

	// Check if it's the player's turn
	if !game.IsPlayersTurn(gameData, playerID) {
		renderGameBoard(c, gameID)
		return
	}

	// Check if cell is empty
	if gameData.Board[row][col] != "" {
		renderGameBoard(c, gameID)
		return
	}

	// Make the move
	gameData.Board[row][col] = player.Emoji
	gameData.MoveCount++

	// Check for winner
	winnerID := game.CheckWinner(gameData)
	if winnerID != "" {
		gameData.Status = models.GameStatusFinished
		gameData.Winner = winnerID

		// Broadcast winner event
		events.BroadcastGameEvent(gameID, models.GameEvent{
			Type:   "game_winner",
			GameID: gameID,
			Data: map[string]interface{}{
				"board":    gameData.Board,
				"winner":   winnerID,
				"emoji":    gameData.Players[winnerID].Emoji,
				"playerID": playerID,
				"row":      row,
				"col":      col,
			},
		})

		// Send personalized game status updates to each player
		events.BroadcastPersonalizedGameStatus(gameID, gameData)
	} else if game.IsBoardFull(gameData) {
		gameData.Status = models.GameStatusDraw

		// Broadcast draw event
		events.BroadcastGameEvent(gameID, models.GameEvent{
			Type:   "game_draw",
			GameID: gameID,
			Data: map[string]interface{}{
				"board":    gameData.Board,
				"playerID": playerID,
				"row":      row,
				"col":      col,
			},
		})

		// Send personalized game status updates to each player
		events.BroadcastPersonalizedGameStatus(gameID, gameData)
	} else {
		// Switch turns
		gameData.CurrentTurn = (gameData.CurrentTurn + 1) % 2

		// Broadcast move event
		events.BroadcastGameEvent(gameID, models.GameEvent{
			Type:   "move",
			GameID: gameID,
			Data: map[string]interface{}{
				"board":      gameData.Board,
				"playerID":   playerID,
				"emoji":      player.Emoji,
				"row":        row,
				"col":        col,
				"nextTurn":   gameData.CurrentTurn,
				"nextPlayer": game.GetCurrentPlayerID(gameData),
			},
		})

		// Send personalized game status updates to each player
		events.BroadcastPersonalizedGameStatus(gameID, gameData)
	}

	renderGameBoard(c, gameID)
}

func GameResetHandler(c *gin.Context) {
	if c.GetHeader("HX-Request") != "true" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HTMX request required"})
		return
	}

	gameID := c.Param("id")
	gameData := game.GetGame(gameID)
	if gameData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Reset all game state
	gameData.Board = models.GameBoard{}
	gameData.Status = models.GameStatusActive
	gameData.Winner = ""
	gameData.MoveCount = 0
	gameData.CurrentTurn = 0

	// Broadcast reset event to all subscribers
	events.BroadcastGameEvent(gameID, models.GameEvent{
		Type:   "reset",
		GameID: gameID,
		Data: map[string]interface{}{
			"board": gameData.Board,
		},
	})

	// Send personalized game status updates to each player
	events.BroadcastPersonalizedGameStatus(gameID, gameData)

	renderGameBoard(c, gameID)
}

func renderGameBoard(c *gin.Context, gameID string) {
	gameData := game.GetGame(gameID)
	if gameData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	response := `<div id="game-board" class="game-board">`

	for row := 0; row < 3; row++ {
		response += `<div class="game-row">`
		for col := 0; col < 3; col++ {
			cellValue := gameData.Board[row][col]
			response += fmt.Sprintf(`<div class="game-cell" hx-post="/api/game/%s/move/%d/%d" hx-target="#game-board" hx-swap="outerHTML">%s</div>`, gameID, row, col, cellValue)
		}
		response += `</div>`
	}

	response += `</div>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, response)
}


func GameSSEHandler(c *gin.Context) {
	gameID := c.Param("id")

	// Validate game exists
	gameData := game.GetGame(gameID)
	if gameData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create subscriber
	subscriber := events.CreateGameSubscriber(gameID, c.Request.Context())
	defer events.RemoveGameSubscriber(subscriber)

	// Send initial game state
	sendInitialGameState(c, gameData)

	// Listen for events
	for {
		select {
		case event := <-subscriber.Channel:
			sendSSEEvent(c, event)
		case <-subscriber.Context.Done():
			return
		}
	}
}

func sendInitialGameState(c *gin.Context, gameData *models.Game) {
	event := models.GameEvent{
		Type:   "initial",
		GameID: gameData.ID,
		Data:   gameData.Board,
	}
	sendSSEEvent(c, event)
}

func sendSSEEvent(c *gin.Context, event models.GameEvent) {
	var eventData string

	switch event.Type {
	case "move", "reset", "game_winner", "game_draw":
		// Extract board from the data map
		dataMap, ok := event.Data.(map[string]interface{})
		if !ok {
			return
		}
		board, ok := dataMap["board"].(models.GameBoard)
		if !ok {
			return
		}
		eventData = renderGameBoardHTML(event.GameID, board)

		fmt.Fprintf(c.Writer, "event: %s\n", event.Type)
		fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)

	case "game_status":
		// Extract game status data
		dataMap, ok := event.Data.(map[string]interface{})
		if !ok {
			return
		}
		gameID, _ := dataMap["gameID"].(string)
		gameData, _ := dataMap["game"].(*models.Game)

		// Get playerID from the current request context
		playerID := getPlayerIDFromContext(c)

		eventData = renderGameStatusHTML(gameID, playerID, gameData)

		fmt.Fprintf(c.Writer, "event: %s\n", event.Type)
		fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)

	case "initial":
		// For initial event, data should still be GameBoard directly
		board, ok := event.Data.(models.GameBoard)
		if !ok {
			return
		}
		eventData = renderGameBoardHTML(event.GameID, board)

		fmt.Fprintf(c.Writer, "event: %s\n", event.Type)
		fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)

	case "player_join":
		fmt.Fprintf(c.Writer, "event: player_join\n")
		fmt.Fprintf(c.Writer, "data: Player joined game\n\n")

	case "game_ready":
		// This triggers redirect to game page for waiting players
		fmt.Fprintf(c.Writer, "event: game_ready\n")
		fmt.Fprintf(c.Writer, "data: Game is ready\n\n")
	}

	c.Writer.Flush()
}

func renderGameBoardHTML(gameID string, board models.GameBoard) string {
	response := `<div id="game-board" class="game-board">`

	for row := 0; row < 3; row++ {
		response += `<div class="game-row">`
		for col := 0; col < 3; col++ {
			cellValue := board[row][col]
			response += fmt.Sprintf(`<div class="game-cell" hx-post="/api/game/%s/move/%d/%d" hx-target="#game-board" hx-swap="outerHTML">%s</div>`, gameID, row, col, cellValue)
		}
		response += `</div>`
	}

	response += `</div>`
	return response
}

func renderGameStatusHTML(gameID, playerID string, gameData *models.Game) string {
	if gameData == nil {
		return `<div id="game-status"></div>`
	}

	response := `<div id="game-status">`

	// Turn indicator for active games
	if game.IsGameActive(gameData) {
		currentTurnPlayerID := game.GetCurrentPlayerID(gameData)
		if currentTurnPlayerID != "" {
			currentPlayer := gameData.Players[currentTurnPlayerID]
			isPlayersTurnValue := game.IsPlayersTurn(gameData, playerID)

			response += `<div class="turn-indicator">`
			if isPlayersTurnValue {
				response += fmt.Sprintf(`<span>🎯 Your turn! (%s)</span>`, currentPlayer.Emoji)
			} else {
				response += fmt.Sprintf(`<span>%s's turn</span>`, currentPlayer.Emoji)
			}
			response += `</div>`
		}
	}

	// Game result for finished games
	if game.IsGameFinished(gameData) {
		if gameData.Winner != "" {
			winner := gameData.Players[gameData.Winner]
			response += fmt.Sprintf(`<div class="game-result winner">🏆 %s wins!</div>`, winner.Emoji)
		} else if gameData.Status == models.GameStatusDraw {
			response += `<div class="game-result draw">🤝 It's a draw!</div>`
		}
	}

	response += `</div>`
	return response
}
