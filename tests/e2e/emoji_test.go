package e2e

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmojiSelection(t *testing.T) {
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

	t.Run("Player must select emoji before accessing game", func(t *testing.T) {
		page, err := browser.NewPage()
		require.NoError(t, err)
		defer page.Close()

		// Create new game
		_, err = page.Goto(server.URL)
		require.NoError(t, err)

		err = page.Click("a:text('New Game')")
		require.NoError(t, err)

		// Should redirect to emoji selection page
		page.WaitForURL("**/game/**/select-emoji")
		url := page.URL()
		assert.Contains(t, url, "/select-emoji", "Should redirect to emoji selection page")

		// Should see emoji selection interface
		emojiGrid := page.Locator(".emoji-grid")
		err = emojiGrid.WaitFor()
		require.NoError(t, err)

		// Should have predefined emojis
		emojis, err := page.Locator(".emoji-option").All()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(emojis), 8, "Should have at least 8 emoji options")

		// Select first emoji
		err = page.Locator(".emoji-option").First().Click()
		require.NoError(t, err)

		// First player should stay in waiting state on emoji selection page
		page.WaitForSelector(".waiting-state")
		finalURL := page.URL()
		assert.Contains(t, finalURL, "/select-emoji", "First player should stay on emoji selection in waiting state")
	})

	t.Run("Improved game flow - Player 1 waits, Player 2 joins, simultaneous entry", func(t *testing.T) {
		// Create two browser contexts
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

		// User A creates game and selects emoji
		t.Log("User A: Creating game and selecting emoji...")
		_, err = userAPage.Goto(server.URL)
		require.NoError(t, err)

		err = userAPage.Click("a:text('New Game')")
		require.NoError(t, err)

		userAPage.WaitForURL("**/game/**/select-emoji")
		
		// User A selects first emoji (🐱)
		err = userAPage.Click(".emoji-option:nth-child(1)")
		require.NoError(t, err)

		// User A should stay on emoji selection page in waiting state
		userAPage.WaitForSelector(".waiting-state")
		
		// Verify User A sees sharing UI
		shareSection, err := userAPage.Locator(".game-sharing").IsVisible()
		require.NoError(t, err)
		assert.True(t, shareSection, "User A should see game sharing section")

		// Verify waiting message shows selected emoji
		waitingMessage, err := userAPage.Locator(".waiting-message").TextContent()
		require.NoError(t, err)
		assert.Contains(t, waitingMessage, "🐱", "Waiting message should show User A's emoji")
		assert.Contains(t, waitingMessage, "Waiting for opponent", "Should show waiting message")

		// Get game URL for User B
		gameURL, err := userAPage.Locator(".url-input").GetAttribute("value")
		require.NoError(t, err)
		require.NotEmpty(t, gameURL)

		// User B joins same game
		t.Log("User B: Joining game via shared URL...")
		_, err = userBPage.Goto(gameURL)
		require.NoError(t, err)

		// Should be redirected to emoji selection
		userBPage.WaitForURL("**/select-emoji")

		// User B should NOT see sharing UI
		shareVisible, err := userBPage.Locator(".game-sharing").IsVisible()
		require.NoError(t, err)
		assert.False(t, shareVisible, "User B should not see game sharing section")

		// First emoji should be disabled/unavailable for User B
		firstEmojiDisabled, err := userBPage.Locator(".emoji-option:nth-child(1)").GetAttribute("disabled")
		require.NoError(t, err)
		assert.NotNil(t, firstEmojiDisabled, "First emoji should be disabled for User B")

		// User B selects second emoji (🚀)
		err = userBPage.Click(".emoji-option:nth-child(2)")
		require.NoError(t, err)

		// Both users should be redirected to game page simultaneously
		userBPage.WaitForURL("**/game/**")
		userAPage.WaitForURL("**/game/**")

		// Verify both players are on the same game
		gameIDFromA := extractGameID(userAPage.URL())
		gameIDFromB := extractGameID(userBPage.URL())
		assert.Equal(t, gameIDFromA, gameIDFromB, "Both players should be in the same game")

		// Verify both players see both emojis in player indicator
		userAIndicator, err := userAPage.Locator(".players-display").TextContent()
		require.NoError(t, err)
		assert.Contains(t, userAIndicator, "🐱", "User A should see their emoji")
		assert.Contains(t, userAIndicator, "🚀", "User A should see User B's emoji")

		userBIndicator, err := userBPage.Locator(".players-display").TextContent()
		require.NoError(t, err)
		assert.Contains(t, userBIndicator, "🐱", "User B should see User A's emoji")
		assert.Contains(t, userBIndicator, "🚀", "User B should see their emoji")

		// Verify game page does NOT have sharing UI
		gameShareVisible, err := userAPage.Locator(".game-sharing").IsVisible()
		require.NoError(t, err)
		assert.False(t, gameShareVisible, "Game page should not have sharing UI")
	})

	t.Run("Emojis appear in game cells instead of X", func(t *testing.T) {
		// Create two browser contexts
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

		// Setup game with two players having different emojis
		_, err = userAPage.Goto(server.URL)
		require.NoError(t, err)

		err = userAPage.Click("a:text('New Game')")
		require.NoError(t, err)

		userAPage.WaitForURL("**/select-emoji")
		err = userAPage.Click(".emoji-option:nth-child(1)") // 🐱
		require.NoError(t, err)

		userAPage.WaitForURL("**/game/**")
		gameURL := userAPage.URL()

		_, err = userBPage.Goto(gameURL)
		require.NoError(t, err)

		userBPage.WaitForURL("**/select-emoji")
		err = userBPage.Click(".emoji-option:nth-child(2)") // 🚀
		require.NoError(t, err)

		userBPage.WaitForURL("**/game/**")

		// User A makes first move
		t.Log("User A making move...")
		err = userAPage.Locator(".game-cell").First().Click()
		require.NoError(t, err)

		// Wait for move to be processed
		_, err = userAPage.WaitForFunction(`document.querySelector('.game-cell').textContent === '🐱'`, nil)
		require.NoError(t, err)

		// Verify User A sees their emoji in the cell
		userAFirstCell, err := userAPage.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🐱", userAFirstCell, "User A should see their emoji in the cell")

		// Verify User B also sees User A's emoji (real-time sync)
		time.Sleep(1 * time.Second) // Give SSE time to sync
		userBFirstCell, err := userBPage.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🐱", userBFirstCell, "User B should see User A's emoji in the cell")

		// User B makes second move
		t.Log("User B making move...")
		err = userBPage.Locator(".game-cell").Nth(4).Click() // Center cell
		require.NoError(t, err)

		// Wait for move to be processed
		_, err = userBPage.WaitForFunction(`document.querySelectorAll('.game-cell')[4].textContent === '🚀'`, nil)
		require.NoError(t, err)

		// Verify both players see both emojis in their respective cells
		userACenterCell, err := userAPage.Locator(".game-cell").Nth(4).TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🚀", userACenterCell, "User A should see User B's emoji in center cell")

		userBCenterCell, err := userBPage.Locator(".game-cell").Nth(4).TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🚀", userBCenterCell, "User B should see their emoji in center cell")
	})

	t.Run("Emoji selection persists after page refresh", func(t *testing.T) {
		// Create two browser contexts for 2-player flow
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

		// User A creates game and selects emoji
		_, err = userAPage.Goto(server.URL)
		require.NoError(t, err)

		err = userAPage.Click("a:text('New Game')")
		require.NoError(t, err)

		userAPage.WaitForURL("**/select-emoji")
		err = userAPage.Click(".emoji-option:nth-child(1)") // 🐱
		require.NoError(t, err)

		// User A should be in waiting state
		userAPage.WaitForSelector(".waiting-state")

		// Get game URL for User B
		gameURL, err := userAPage.Locator(".url-input").GetAttribute("value")
		require.NoError(t, err)

		// User B joins and selects emoji to start game
		_, err = userBPage.Goto(gameURL)
		require.NoError(t, err)

		userBPage.WaitForURL("**/select-emoji")
		err = userBPage.Click(".emoji-option:nth-child(2)") // 🚀
		require.NoError(t, err)

		// Both users should enter the game
		userAPage.WaitForURL("**/game/**")
		userBPage.WaitForURL("**/game/**")

		// User A makes a move
		err = userAPage.Locator(".game-cell").First().Click()
		require.NoError(t, err)
		_, err = userAPage.WaitForFunction(`document.querySelector('.game-cell').textContent === '🐱'`, nil)
		require.NoError(t, err)

		// Refresh User A's page
		_, err = userAPage.Reload()
		require.NoError(t, err)

		// Should not redirect to emoji selection
		finalURL := userAPage.URL()
		assert.NotContains(t, finalURL, "/select-emoji", "Should not redirect to emoji selection after refresh")

		// Should still see the emoji in the cell and player display
		firstCell, err := userAPage.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🐱", firstCell, "Should persist emoji in game cell after refresh")

		playerDisplay, err := userAPage.Locator(".players-display").TextContent()
		require.NoError(t, err)
		assert.Contains(t, playerDisplay, "🐱", "Should persist emoji in player display after refresh")
		assert.Contains(t, playerDisplay, "🚀", "Should persist both player emojis after refresh")
	})

	t.Run("Game limits to 2 players maximum", func(t *testing.T) {
		// Create three browser contexts
		userAContext, err := browser.NewContext()
		require.NoError(t, err)
		defer userAContext.Close()

		userBContext, err := browser.NewContext()
		require.NoError(t, err)
		defer userBContext.Close()

		userCContext, err := browser.NewContext()
		require.NoError(t, err)
		defer userCContext.Close()

		userAPage, err := userAContext.NewPage()
		require.NoError(t, err)

		userBPage, err := userBContext.NewPage()
		require.NoError(t, err)

		userCPage, err := userCContext.NewPage()
		require.NoError(t, err)

		// User A creates game and selects emoji
		t.Log("User A: Creating game...")
		_, err = userAPage.Goto(server.URL)
		require.NoError(t, err)

		err = userAPage.Click("a:text('New Game')")
		require.NoError(t, err)

		userAPage.WaitForURL("**/select-emoji")
		err = userAPage.Click(".emoji-option:nth-child(1)") // 🐱
		require.NoError(t, err)

		// Get game URL
		gameURL, err := userAPage.Locator(".url-input").GetAttribute("value")
		require.NoError(t, err)

		// User B joins and selects emoji
		t.Log("User B: Joining game...")
		_, err = userBPage.Goto(gameURL)
		require.NoError(t, err)

		userBPage.WaitForURL("**/select-emoji")
		err = userBPage.Click(".emoji-option:nth-child(2)") // 🚀
		require.NoError(t, err)

		// Both should be in game now
		userAPage.WaitForURL("**/game/**")
		userBPage.WaitForURL("**/game/**")

		// User C tries to join - should be rejected
		t.Log("User C: Attempting to join full game...")
		_, err = userCPage.Goto(gameURL)
		require.NoError(t, err)

		// Should see game full message or be redirected to home
		gameFull, err := userCPage.Locator(".game-full").IsVisible()
		if err == nil && !gameFull {
			// Alternative: check if redirected to home
			url := userCPage.URL()
			assert.NotContains(t, url, "/game/", "Third player should not access game")
		} else {
			assert.True(t, gameFull, "Third player should see game full message")
		}
	})
}