package handlers

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)


type GameBoard [3][3]string

type Player struct {
	ID       string
	Emoji    string
	JoinedAt time.Time
}

type GameStatus string

const (
	GameStatusWaiting  GameStatus = "waiting"  // 1 player, waiting for opponent
	GameStatusReady    GameStatus = "ready"    // 2 players, game can be played
	GameStatusActive   GameStatus = "active"   // Game is being played
	GameStatusFinished GameStatus = "finished" // Game finished with a winner
	GameStatusDraw     GameStatus = "draw"     // Game finished in a draw
	GameStatusFull     GameStatus = "full"     // 2 players, no more joins allowed
)

const MaxPlayersPerGame = 2

type Game struct {
	ID          string
	Board       GameBoard
	Players     map[string]*Player // playerID -> Player
	PlayerOrder []string           // track join order
	Status      GameStatus         // current game status
	CurrentTurn int                // index into PlayerOrder (0 or 1)
	Winner      string             // playerID of winner (if any)
	MoveCount   int                // total moves made
}

type GameEvent struct {
	Type   string      `json:"type"`
	GameID string      `json:"gameId"`
	Data   interface{} `json:"data"`
}

type GameSubscriber struct {
	ID      string
	GameID  string
	Channel chan GameEvent
	Context context.Context
}

var (
	games           = make(map[string]*Game)
	gamesMux        = sync.RWMutex{}
	gameSubscribers = make(map[string][]*GameSubscriber)
	subscribersMux  = sync.RWMutex{}
)

func generateGameID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func createGame() *Game {
	//gamesMux.Lock()
	//defer gamesMux.Unlock()

	id := generateGameID()
	game := &Game{
		ID:          id,
		Board:       GameBoard{},
		Players:     make(map[string]*Player),
		PlayerOrder: make([]string, 0),
		Status:      GameStatusWaiting, // Start in waiting state
	}
	games[id] = game
	return game
}

func getGame(id string) *Game {
	//gamesMux.RLock()
	//defer gamesMux.RUnlock()
	return games[id]
}

// Predefined emoji options
var availableEmojis = []string{"🐱", "🚀", "🎨", "🌟", "🔥", "⚡", "🎮", "🦄", "🎯", "🌈"}

func generatePlayerID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("player_%x", bytes)
}

func getPlayerIDFromContext(c *gin.Context) string {
	// Simple approach: use session cookie or generate new ID
	playerID, err := c.Cookie("player_id")
	if err != nil || playerID == "" {
		playerID = generatePlayerID()
		c.SetCookie("player_id", playerID, 3600*24, "/", "", false, true)
	}
	return playerID
}

func isEmojiAvailable(game *Game, emoji string) bool {
	for _, player := range game.Players {
		if player.Emoji == emoji {
			return false
		}
	}
	return true
}

func isFirstPlayer(game *Game, playerID string) bool {
	return len(game.Players) == 1 && game.Players[playerID] != nil
}

func isGameReady(game *Game) bool {
	return game.Status == GameStatusActive || game.Status == GameStatusFinished || game.Status == GameStatusDraw
}

func isGameActive(game *Game) bool {
	return game.Status == GameStatusActive
}

func isGameFinished(game *Game) bool {
	return game.Status == GameStatusFinished || game.Status == GameStatusDraw
}

func canJoinGame(game *Game) bool {
	return len(game.Players) < MaxPlayersPerGame
}

func getCurrentPlayerID(game *Game) string {
	if !isGameActive(game) || len(game.PlayerOrder) < 2 {
		return ""
	}
	return game.PlayerOrder[game.CurrentTurn]
}

func isPlayersTurn(game *Game, playerID string) bool {
	return isGameActive(game) && getCurrentPlayerID(game) == playerID
}

func checkWinner(game *Game) string {
	board := game.Board

	// Check rows
	for row := 0; row < 3; row++ {
		if board[row][0] != "" && board[row][0] == board[row][1] && board[row][1] == board[row][2] {
			// Find playerID by emoji
			for pID, player := range game.Players {
				if player.Emoji == board[row][0] {
					return pID
				}
			}
		}
	}

	// Check columns
	for col := 0; col < 3; col++ {
		if board[0][col] != "" && board[0][col] == board[1][col] && board[1][col] == board[2][col] {
			// Find playerID by emoji
			for pID, player := range game.Players {
				if player.Emoji == board[0][col] {
					return pID
				}
			}
		}
	}

	// Check main diagonal (top-left to bottom-right)
	if board[0][0] != "" && board[0][0] == board[1][1] && board[1][1] == board[2][2] {
		for pID, player := range game.Players {
			if player.Emoji == board[0][0] {
				return pID
			}
		}
	}

	// Check anti-diagonal (top-right to bottom-left)
	if board[0][2] != "" && board[0][2] == board[1][1] && board[1][1] == board[2][0] {
		for pID, player := range game.Players {
			if player.Emoji == board[0][2] {
				return pID
			}
		}
	}

	return "" // No winner
}

func isBoardFull(game *Game) bool {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if game.Board[row][col] == "" {
				return false
			}
		}
	}
	return true
}

func addPlayerToGame(game *Game, playerID, emoji string) error {
	// Check if game is full
	if len(game.Players) >= MaxPlayersPerGame {
		return fmt.Errorf("game is full")
	}

	// Check if player already in game
	if _, exists := game.Players[playerID]; exists {
		return fmt.Errorf("player already in game")
	}

	if !isEmojiAvailable(game, emoji) {
		return fmt.Errorf("emoji already taken")
	}

	// Check if emoji is in available list
	emojiValid := false
	for _, availableEmoji := range availableEmojis {
		if availableEmoji == emoji {
			emojiValid = true
			break
		}
	}
	if !emojiValid {
		return fmt.Errorf("invalid emoji")
	}

	player := &Player{
		ID:       playerID,
		Emoji:    emoji,
		JoinedAt: time.Now(),
	}

	game.Players[playerID] = player
	game.PlayerOrder = append(game.PlayerOrder, playerID)

	// Update game status based on player count
	if len(game.Players) == 1 {
		game.Status = GameStatusWaiting
	} else if len(game.Players) == MaxPlayersPerGame {
		game.Status = GameStatusActive // Start the game with first player's turn
		game.CurrentTurn = 0           // Player 1 (index 0) goes first
		game.MoveCount = 0
	}

	return nil
}

func HomeHandler(c *gin.Context) {
	data := gin.H{
		"Title": "Tic-Tac-Toe Game",
		"NeedsSSE": false,
		"PageType": "home",
	}

	c.HTML(http.StatusOK, "base.html", data)
}

func NewGameHandler(c *gin.Context) {
	game := createGame()
	c.Redirect(http.StatusSeeOther, "/game/"+game.ID+"/select-emoji")
}

func GamePageHandler(c *gin.Context) {
	gameID := c.Param("id")
	game := getGame(gameID)

	if game == nil {
		c.HTML(http.StatusNotFound, "base.html", gin.H{
			"Title": "Game Not Found",
			"NeedsSSE": false,
			"PageType": "notfound",
		})
		return
	}

	// Check if player has selected emoji
	playerID := getPlayerIDFromContext(c)
	player, playerExists := game.Players[playerID]

	if !playerExists || player.Emoji == "" {
		// Redirect to emoji selection
		c.Redirect(http.StatusSeeOther, "/game/"+gameID+"/select-emoji")
		return
	}

	// Only allow access when game is ready (2 players)
	if !isGameReady(game) {
		// Redirect back to emoji selection (will show waiting state if needed)
		c.Redirect(http.StatusSeeOther, "/game/"+gameID+"/select-emoji")
		return
	}

	// Get player list for display
	var playerEmojis []string
	for _, pID := range game.PlayerOrder {
		if p, exists := game.Players[pID]; exists {
			playerEmojis = append(playerEmojis, p.Emoji)
		}
	}

	// Get current turn information
	currentTurnPlayerID := getCurrentPlayerID(game)
	var currentTurnEmoji string
	if currentTurnPlayerID != "" {
		if currentPlayer, exists := game.Players[currentTurnPlayerID]; exists {
			currentTurnEmoji = currentPlayer.Emoji
		}
	}

	// Get winner information
	var winnerEmoji string
	if game.Winner != "" {
		if winner, exists := game.Players[game.Winner]; exists {
			winnerEmoji = winner.Emoji
		}
	}

	data := gin.H{
		"Title":            "Tic-Tac-Toe Game #" + gameID,
		"GameID":           gameID,
		"PlayerEmojis":     playerEmojis,
		"CurrentPlayer":    player,
		"GameStatus":       game.Status,
		"CurrentTurnEmoji": currentTurnEmoji,
		"IsPlayersTurn":    isPlayersTurn(game, playerID),
		"WinnerEmoji":      winnerEmoji,
		"IsGameActive":     isGameActive(game),
		"IsGameFinished":   isGameFinished(game),
		"NeedsSSE":         true,
		"PageType":         "game",
	}

	c.HTML(http.StatusOK, "base.html", data)
}

func EmojiSelectionHandler(c *gin.Context) {
	gameID := c.Param("id")
	game := getGame(gameID)

	if game == nil {
		c.HTML(http.StatusNotFound, "base.html", gin.H{
			"Title": "Game Not Found",
			"NeedsSSE": false,
			"PageType": "notfound",
		})
		return
	}

	playerID := getPlayerIDFromContext(c)

	// Check if game is full
	if !canJoinGame(game) {
		// Check if this player is already in the game
		if _, exists := game.Players[playerID]; !exists {
			c.HTML(http.StatusOK, "base.html", gin.H{
				"Title": "Game Full",
				"NeedsSSE": false,
				"PageType": "gamefull",
			})
			return
		}
	}

	// If player already has emoji selected
	if player, exists := game.Players[playerID]; exists && player.Emoji != "" {
		// Check if this is the first player and game is still waiting
		if isFirstPlayer(game, playerID) && game.Status == GameStatusWaiting {
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
				"NeedsSSE":       true,
				"PageType":       "emoji",
			}
			c.HTML(http.StatusOK, "base.html", data)
			return
		}

		// If game is ready, redirect to game
		if isGameReady(game) {
			c.Redirect(http.StatusSeeOther, "/game/"+gameID)
			return
		}
	}

	// Get available emojis (not taken by other players)
	var availableEmojiList []map[string]interface{}
	for _, emoji := range availableEmojis {
		available := isEmojiAvailable(game, emoji)
		availableEmojiList = append(availableEmojiList, map[string]interface{}{
			"emoji":     emoji,
			"available": available,
		})
	}

	// Determine if this would be the first player
	wouldBeFirst := len(game.Players) == 0

	data := gin.H{
		"Title":           "Select Your Emoji",
		"GameID":          gameID,
		"AvailableEmojis": availableEmojiList,
		"IsWaitingState":  false,
		"IsFirstPlayer":   wouldBeFirst,
		"NeedsSSE":        true,
		"PageType":        "emoji",
	}

	c.HTML(http.StatusOK, "base.html", data)
}

func EmojiSelectionSubmitHandler(c *gin.Context) {
	gameID := c.Param("id")
	game := getGame(gameID)

	if game == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	playerID := getPlayerIDFromContext(c)
	selectedEmoji := c.PostForm("emoji")

	if selectedEmoji == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No emoji selected"})
		return
	}

	//gamesMux.Lock()
	isFirstPlayerJoining := len(game.Players) == 0
	err := addPlayerToGame(game, playerID, selectedEmoji)
	isGameReadyNow := game.Status == GameStatusActive
	//gamesMux.Unlock()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Broadcast player join event
	broadcastGameEvent(gameID, GameEvent{
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
		broadcastGameEvent(gameID, GameEvent{
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

	game := getGame(gameID)
	if game == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Get player ID and check if player exists
	playerID := getPlayerIDFromContext(c)
	player, exists := game.Players[playerID]
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

	//gamesMux.Lock()
	//defer gamesMux.Unlock()

	// Check if game is finished
	if isGameFinished(game) {
		renderGameBoard(c, gameID)
		return
	}

	// Check if it's the player's turn
	if !isPlayersTurn(game, playerID) {
		renderGameBoard(c, gameID)
		return
	}

	// Check if cell is empty
	if game.Board[row][col] != "" {
		renderGameBoard(c, gameID)
		return
	}

	// Make the move
	game.Board[row][col] = player.Emoji
	game.MoveCount++

	// Check for winner
	winnerID := checkWinner(game)
	if winnerID != "" {
		game.Status = GameStatusFinished
		game.Winner = winnerID

		// Broadcast winner event
		broadcastGameEvent(gameID, GameEvent{
			Type:   "game_winner",
			GameID: gameID,
			Data: map[string]interface{}{
				"board":    game.Board,
				"winner":   winnerID,
				"emoji":    game.Players[winnerID].Emoji,
				"playerID": playerID,
				"row":      row,
				"col":      col,
			},
		})

		// Send personalized game status updates to each player
		broadcastPersonalizedGameStatus(gameID, game)
	} else if isBoardFull(game) {
		game.Status = GameStatusDraw

		// Broadcast draw event
		broadcastGameEvent(gameID, GameEvent{
			Type:   "game_draw",
			GameID: gameID,
			Data: map[string]interface{}{
				"board":    game.Board,
				"playerID": playerID,
				"row":      row,
				"col":      col,
			},
		})

		// Send personalized game status updates to each player
		broadcastPersonalizedGameStatus(gameID, game)
	} else {
		// Switch turns
		game.CurrentTurn = (game.CurrentTurn + 1) % 2

		// Broadcast move event
		broadcastGameEvent(gameID, GameEvent{
			Type:   "move",
			GameID: gameID,
			Data: map[string]interface{}{
				"board":      game.Board,
				"playerID":   playerID,
				"emoji":      player.Emoji,
				"row":        row,
				"col":        col,
				"nextTurn":   game.CurrentTurn,
				"nextPlayer": getCurrentPlayerID(game),
			},
		})

		// Send personalized game status updates to each player
		broadcastPersonalizedGameStatus(gameID, game)
	}

	renderGameBoard(c, gameID)
}

func GameResetHandler(c *gin.Context) {
	if c.GetHeader("HX-Request") != "true" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HTMX request required"})
		return
	}

	gameID := c.Param("id")
	game := getGame(gameID)
	if game == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	//gamesMux.Lock()
	game.Board = GameBoard{}

	// Broadcast reset event to all subscribers
	broadcastGameEvent(gameID, GameEvent{
		Type:   "reset",
		GameID: gameID,
		Data: map[string]interface{}{
			"board": game.Board,
		},
	})
	//gamesMux.Unlock()

	renderGameBoard(c, gameID)
}

func renderGameBoard(c *gin.Context, gameID string) {
	game := getGame(gameID)
	if game == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	response := `<div id="game-board" class="game-board">`

	//gamesMux.RLock()
	for row := 0; row < 3; row++ {
		response += `<div class="game-row">`
		for col := 0; col < 3; col++ {
			cellValue := game.Board[row][col]
			response += fmt.Sprintf(`<div class="game-cell" hx-post="/api/game/%s/move/%d/%d" hx-target="#game-board" hx-swap="outerHTML">%s</div>`, gameID, row, col, cellValue)
		}
		response += `</div>`
	}
	//gamesMux.RUnlock()

	response += `</div>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, response)
}

func generateSubscriberID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func createGameSubscriber(gameID string, ctx context.Context) *GameSubscriber {
	subscriber := &GameSubscriber{
		ID:      generateSubscriberID(),
		GameID:  gameID,
		Channel: make(chan GameEvent, 10), // Buffer for events
		Context: ctx,
	}

	//subscribersMux.Lock()
	gameSubscribers[gameID] = append(gameSubscribers[gameID], subscriber)
	//subscribersMux.Unlock()

	return subscriber
}

func removeGameSubscriber(subscriber *GameSubscriber) {
	//subscribersMux.Lock()
	//defer subscribersMux.Unlock()

	subscribers, exists := gameSubscribers[subscriber.GameID]
	if !exists {
		return
	}

	for i, sub := range subscribers {
		if sub.ID == subscriber.ID {
			gameSubscribers[subscriber.GameID] = append(subscribers[:i], subscribers[i+1:]...)
			close(sub.Channel)
			break
		}
	}

	if len(gameSubscribers[subscriber.GameID]) == 0 {
		delete(gameSubscribers, subscriber.GameID)
	}
}

func broadcastGameEvent(gameID string, event GameEvent) {
	//subscribersMux.RLock()
	subscribers, exists := gameSubscribers[gameID]
	//subscribersMux.RUnlock()

	if !exists {
		return
	}

	for _, subscriber := range subscribers {
		select {
		case subscriber.Channel <- event:
		case <-subscriber.Context.Done():
			go removeGameSubscriber(subscriber)
		default:
			// Channel full, skip this subscriber
		}
	}
}

func broadcastPersonalizedGameStatus(gameID string, game *Game) {
	//subscribersMux.RLock()
	subscribers, exists := gameSubscribers[gameID]
	//subscribersMux.RUnlock()

	if !exists {
		return
	}

	// For each subscriber, we need to determine their playerID and send personalized status
	// Since we don't have direct access to playerID per subscriber, we'll send to all players
	// and let the SSE handler figure out the playerID from the request context
	for _, subscriber := range subscribers {
		event := GameEvent{
			Type:   "game_status",
			GameID: gameID,
			Data: map[string]interface{}{
				"gameID": gameID,
				"game":   game,
			},
		}

		select {
		case subscriber.Channel <- event:
		case <-subscriber.Context.Done():
			go removeGameSubscriber(subscriber)
		default:
			// Channel full, skip this subscriber
		}
	}
}

func GameSSEHandler(c *gin.Context) {
	gameID := c.Param("id")

	// Validate game exists
	game := getGame(gameID)
	if game == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Create subscriber
	subscriber := createGameSubscriber(gameID, c.Request.Context())
	defer removeGameSubscriber(subscriber)

	// Send initial game state
	sendInitialGameState(c, game)

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

func sendInitialGameState(c *gin.Context, game *Game) {
	event := GameEvent{
		Type:   "initial",
		GameID: game.ID,
		Data:   game.Board,
	}
	sendSSEEvent(c, event)
}

func sendSSEEvent(c *gin.Context, event GameEvent) {
	var eventData string

	switch event.Type {
	case "move", "reset", "game_winner", "game_draw":
		// Extract board from the data map
		dataMap, ok := event.Data.(map[string]interface{})
		if !ok {
			return
		}
		board, ok := dataMap["board"].(GameBoard)
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
		game, _ := dataMap["game"].(*Game)

		// Get playerID from the current request context
		playerID := getPlayerIDFromContext(c)

		eventData = renderGameStatusHTML(gameID, playerID, game)

		fmt.Fprintf(c.Writer, "event: %s\n", event.Type)
		fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)

	case "initial":
		// For initial event, data should still be GameBoard directly
		board, ok := event.Data.(GameBoard)
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

func renderGameBoardHTML(gameID string, board GameBoard) string {
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

func renderGameStatusHTML(gameID, playerID string, game *Game) string {
	if game == nil {
		return `<div id="game-status"></div>`
	}

	response := `<div id="game-status">`

	// Turn indicator for active games
	if isGameActive(game) {
		currentTurnPlayerID := getCurrentPlayerID(game)
		if currentTurnPlayerID != "" {
			currentPlayer := game.Players[currentTurnPlayerID]
			isPlayersTurnValue := isPlayersTurn(game, playerID)

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
	if isGameFinished(game) {
		if game.Winner != "" {
			winner := game.Players[game.Winner]
			response += fmt.Sprintf(`<div class="game-result winner">🏆 %s wins!</div>`, winner.Emoji)
		} else if game.Status == GameStatusDraw {
			response += `<div class="game-result draw">🤝 It's a draw!</div>`
		}
	}

	response += `</div>`
	return response
}
