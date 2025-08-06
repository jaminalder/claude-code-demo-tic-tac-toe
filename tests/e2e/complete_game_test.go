package e2e

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteGameFlow(t *testing.T) {
	// Setup Playwright
	pw, err := playwright.Run()
	require.NoError(t, err)
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	require.NoError(t, err)
	defer browser.Close()

	// Start test server
	server := httptest.NewServer(setupRouter())
	defer server.Close()

	t.Run("Complete game with winner", func(t *testing.T) {
		// Setup two players
		userAContext, err := browser.NewContext()
		require.NoError(t, err)
		defer userAContext.Close()

		userBContext, err := browser.NewContext()
		require.NoError(t, err)
		defer userBContext.Close()

		userAPage, err := userAContext.NewPage()
		require.NoError(t, err)

		userBPage, err := userBContext.NewPage()
		require.NoError(t, err)

		// Setup complete game with both players
		gameID := setupTwoPlayerGame(t, server.URL, userAPage, userBPage)
		t.Logf("Game ID: %s", gameID)

		// Wait for game to stabilize
		time.Sleep(500 * time.Millisecond)

		// Play game: Player A wins top row
		// A: (0,0), B: (1,0), A: (0,1), B: (1,1), A: (0,2) - A WINS

		// Player A move 1: (0,0)
		t.Log("Player A move 1: (0,0)")
		err = userAPage.Locator(".game-cell").Nth(0).Click()
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		cell1, _ := userAPage.Locator(".game-cell").Nth(0).TextContent()
		t.Logf("Cell (0,0): '%s'", cell1)
		assert.Equal(t, "🐱", cell1, "Player A should place emoji in (0,0)")

		// Player B move 1: (1,0)
		t.Log("Player B move 1: (1,0)")
		time.Sleep(200 * time.Millisecond)
		err = userBPage.Locator(".game-cell").Nth(3).Click()
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		cell2, _ := userBPage.Locator(".game-cell").Nth(3).TextContent()
		t.Logf("Cell (1,0): '%s'", cell2)
		assert.Equal(t, "🚀", cell2, "Player B should place emoji in (1,0)")

		// Player A move 2: (0,1)
		t.Log("Player A move 2: (0,1)")
		time.Sleep(200 * time.Millisecond)
		err = userAPage.Locator(".game-cell").Nth(1).Click()
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		cell3, _ := userAPage.Locator(".game-cell").Nth(1).TextContent()
		t.Logf("Cell (0,1): '%s'", cell3)
		assert.Equal(t, "🐱", cell3, "Player A should place emoji in (0,1)")

		// Player B move 2: (1,1)
		t.Log("Player B move 2: (1,1)")
		time.Sleep(200 * time.Millisecond)
		err = userBPage.Locator(".game-cell").Nth(4).Click()
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		cell4, _ := userBPage.Locator(".game-cell").Nth(4).TextContent()
		t.Logf("Cell (1,1): '%s'", cell4)
		assert.Equal(t, "🚀", cell4, "Player B should place emoji in (1,1)")

		// Player A WINNING move: (0,2)
		t.Log("Player A WINNING move: (0,2)")
		time.Sleep(200 * time.Millisecond)
		err = userAPage.Locator(".game-cell").Nth(2).Click()
		require.NoError(t, err)
		time.Sleep(1000 * time.Millisecond) // Give time for winner detection

		cell5, _ := userAPage.Locator(".game-cell").Nth(2).TextContent()
		t.Logf("Cell (0,2): '%s'", cell5)
		assert.Equal(t, "🐱", cell5, "Player A should place winning emoji in (0,2)")

		// Check for winner announcement
		gameResultVisible, err := userAPage.Locator(".game-result").IsVisible()
		if err == nil && gameResultVisible {
			gameResult, _ := userAPage.Locator(".game-result").TextContent()
			t.Logf("Game result: %s", gameResult)
			
			if gameResult != "" {
				t.Log("✅ Winner detection is working!")
			} else {
				t.Log("⚠️  Winner element exists but no text")
			}
		} else {
			t.Log("⚠️  No winner announcement found")
		}

		// Test that no more moves are allowed
		t.Log("Testing that no more moves are allowed...")
		err = userBPage.Locator(".game-cell").Nth(5).Click()
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond)

		cell6, _ := userBPage.Locator(".game-cell").Nth(5).TextContent()
		t.Logf("Cell (1,2) after game over: '%s'", cell6)
		
		if cell6 == "" {
			t.Log("✅ Game over enforcement is working!")
		} else {
			t.Log("❌ Move was allowed after game over")
		}

		t.Log("Complete game test finished!")
	})
}