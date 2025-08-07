package models

import (
	"context"
	"time"
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

// Predefined emoji options
var AvailableEmojis = []string{"🐱", "🚀", "🎨", "🌟", "🔥", "⚡", "🎮", "🦄", "🎯", "🌈"}