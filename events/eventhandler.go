package events

import (
	"context"
	"crypto/rand"
	"fmt"

	"htmx-go-app/models"
)

// Global subscriber management
var gameSubscribers = make(map[string][]*models.GameSubscriber)

// generateSubscriberID creates a unique subscriber identifier
func generateSubscriberID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// CreateGameSubscriber creates and registers a new subscriber for a game
func CreateGameSubscriber(gameID string, ctx context.Context) *models.GameSubscriber {
	subscriber := &models.GameSubscriber{
		ID:      generateSubscriberID(),
		GameID:  gameID,
		Channel: make(chan models.GameEvent, 10), // Buffer for events
		Context: ctx,
	}

	gameSubscribers[gameID] = append(gameSubscribers[gameID], subscriber)

	return subscriber
}

// RemoveGameSubscriber removes a subscriber and cleans up resources
func RemoveGameSubscriber(subscriber *models.GameSubscriber) {
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

// BroadcastGameEvent sends an event to all subscribers of a game
func BroadcastGameEvent(gameID string, event models.GameEvent) {
	subscribers, exists := gameSubscribers[gameID]

	if !exists {
		return
	}

	for _, subscriber := range subscribers {
		select {
		case subscriber.Channel <- event:
		case <-subscriber.Context.Done():
			go RemoveGameSubscriber(subscriber)
		default:
			// Channel full, skip this subscriber
		}
	}
}

// BroadcastPersonalizedGameStatus sends personalized game status to all subscribers
func BroadcastPersonalizedGameStatus(gameID string, game *models.Game) {
	subscribers, exists := gameSubscribers[gameID]

	if !exists {
		return
	}

	// For each subscriber, we need to determine their playerID and send personalized status
	// Since we don't have direct access to playerID per subscriber, we'll send to all players
	// and let the SSE handler figure out the playerID from the request context
	for _, subscriber := range subscribers {
		event := models.GameEvent{
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
			go RemoveGameSubscriber(subscriber)
		default:
			// Channel full, skip this subscriber
		}
	}
}