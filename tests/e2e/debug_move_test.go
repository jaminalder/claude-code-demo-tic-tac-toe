package e2e

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
)

func TestDebugMove(t *testing.T) {
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

	t.Run("Debug single move", func(t *testing.T) {
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
		time.Sleep(1000 * time.Millisecond)

		// Try to make a move with Player A
		t.Log("Player A attempting to click first cell...")
		err = userAPage.Locator(".game-cell").First().Click()
		require.NoError(t, err)

		// Wait to see what happens
		time.Sleep(2000 * time.Millisecond)

		// Check if move was successful
		firstCellContent, err := userAPage.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)

		t.Logf("First cell content after move: '%s'", firstCellContent)

		if firstCellContent == "🐱" {
			t.Log("✅ Move was successful!")
		} else {
			t.Log("❌ Move failed - cell is still empty")
		}
	})
}