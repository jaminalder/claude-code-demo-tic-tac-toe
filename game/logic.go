package game

import "htmx-go-app/models"

// CheckWinner returns the playerID of the winner, or empty string if no winner
func CheckWinner(game *models.Game) string {
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

// IsBoardFull checks if all cells on the board are filled
func IsBoardFull(game *models.Game) bool {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if game.Board[row][col] == "" {
				return false
			}
		}
	}
	return true
}

// IsGameActive returns true if the game is currently active
func IsGameActive(game *models.Game) bool {
	return game.Status == models.GameStatusActive
}

// IsGameFinished returns true if the game has finished (winner or draw)
func IsGameFinished(game *models.Game) bool {
	return game.Status == models.GameStatusFinished || game.Status == models.GameStatusDraw
}

// IsGameReady returns true if the game is ready to be played
func IsGameReady(game *models.Game) bool {
	return game.Status == models.GameStatusActive || game.Status == models.GameStatusFinished || game.Status == models.GameStatusDraw
}

// CanJoinGame returns true if the game can accept more players
func CanJoinGame(game *models.Game) bool {
	return len(game.Players) < models.MaxPlayersPerGame
}

// GetCurrentPlayerID returns the ID of the player whose turn it is
func GetCurrentPlayerID(game *models.Game) string {
	if !IsGameActive(game) || len(game.PlayerOrder) < 2 {
		return ""
	}
	return game.PlayerOrder[game.CurrentTurn]
}

// IsPlayersTurn returns true if it's the specified player's turn
func IsPlayersTurn(game *models.Game, playerID string) bool {
	return IsGameActive(game) && GetCurrentPlayerID(game) == playerID
}

// IsEmojiAvailable returns true if the emoji is not already taken by another player
func IsEmojiAvailable(game *models.Game, emoji string) bool {
	for _, player := range game.Players {
		if player.Emoji == emoji {
			return false
		}
	}
	return true
}

// IsFirstPlayer returns true if the given player is the first (and only) player in the game
func IsFirstPlayer(game *models.Game, playerID string) bool {
	return len(game.Players) == 1 && game.Players[playerID] != nil
}