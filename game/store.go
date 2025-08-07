package game

import (
	"crypto/rand"
	"fmt"
	"time"

	"htmx-go-app/models"
)

// Global game storage
var games = make(map[string]*models.Game)

// generateGameID creates a unique game identifier
func generateGameID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// GeneratePlayerID creates a unique player identifier
func GeneratePlayerID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("player_%x", bytes)
}

// CreateGame creates a new game and stores it
func CreateGame() *models.Game {
	id := generateGameID()
	game := &models.Game{
		ID:          id,
		Board:       models.GameBoard{},
		Players:     make(map[string]*models.Player),
		PlayerOrder: make([]string, 0),
		Status:      models.GameStatusWaiting, // Start in waiting state
	}
	games[id] = game
	return game
}

// GetGame retrieves a game by ID
func GetGame(id string) *models.Game {
	return games[id]
}

// AddPlayerToGame adds a player with the given emoji to the game
func AddPlayerToGame(game *models.Game, playerID, emoji string) error {
	// Check if game is full
	if len(game.Players) >= models.MaxPlayersPerGame {
		return fmt.Errorf("game is full")
	}

	// Check if player already in game
	if _, exists := game.Players[playerID]; exists {
		return fmt.Errorf("player already in game")
	}

	if !IsEmojiAvailable(game, emoji) {
		return fmt.Errorf("emoji already taken")
	}

	// Check if emoji is in available list
	emojiValid := false
	for _, availableEmoji := range models.AvailableEmojis {
		if availableEmoji == emoji {
			emojiValid = true
			break
		}
	}
	if !emojiValid {
		return fmt.Errorf("invalid emoji")
	}

	player := &models.Player{
		ID:       playerID,
		Emoji:    emoji,
		JoinedAt: time.Now(),
	}

	game.Players[playerID] = player
	game.PlayerOrder = append(game.PlayerOrder, playerID)

	// Update game status based on player count
	if len(game.Players) == 1 {
		game.Status = models.GameStatusWaiting
	} else if len(game.Players) == models.MaxPlayersPerGame {
		game.Status = models.GameStatusActive // Start the game with first player's turn
		game.CurrentTurn = 0                  // Player 1 (index 0) goes first
		game.MoveCount = 0
	}

	return nil
}