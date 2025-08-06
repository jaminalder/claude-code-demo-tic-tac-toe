package e2e

import (
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"htmx-go-app/handlers"

	"github.com/gin-gonic/gin"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.LoadHTMLGlob("../../templates/**/*")
	r.Static("/static", "../../static")

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

	return r
}

func extractGameID(gameURL string) string {
	re := regexp.MustCompile(`/game/([a-f0-9]+)`)
	matches := re.FindStringSubmatch(gameURL)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func TestMultiUserGameplay(t *testing.T) {
	// Setup Playwright
	pw, err := playwright.Run()
	require.NoError(t, err)
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Set to true for CI
	})
	require.NoError(t, err)
	defer browser.Close()

	// Start test server
	server := httptest.NewServer(setupRouter())
	defer server.Close()

	// Create two separate browser contexts (simulate different users)
	userAContext, err := browser.NewContext()
	require.NoError(t, err)
	defer userAContext.Close()

	userBContext, err := browser.NewContext()
	require.NoError(t, err)
	defer userBContext.Close()

	// Create pages for each user
	userAPage, err := userAContext.NewPage()
	require.NoError(t, err)

	userBPage, err := userBContext.NewPage()
	require.NoError(t, err)

	t.Run("Two users play same game with real-time sync", func(t *testing.T) {
		// Step 1: User A creates new game and selects emoji
		t.Log("User A: Creating new game and selecting emoji...")
		_, err = userAPage.Goto(server.URL)
		require.NoError(t, err)

		err = userAPage.Click("a:text('New Game')")
		require.NoError(t, err)

		userAPage.WaitForURL("**/game/**/select-emoji")

		// User A selects first emoji
		err = userAPage.Click(".emoji-option:nth-child(1)")
		require.NoError(t, err)

		// User A should be in waiting state, get game URL from sharing section
		userAPage.WaitForSelector(".waiting-state")
		gameURL, err := userAPage.Locator(".url-input").GetAttribute("value")
		require.NoError(t, err)

		// Extract game ID from URL
		gameID := extractGameID(gameURL)
		require.NotEmpty(t, gameID, "Game ID should be extracted from URL")
		t.Logf("Created game: %s", gameID)

		// Step 2: User B joins same game using shared URL and selects emoji
		t.Log("User B: Joining game via shared URL and selecting emoji...")
		_, err = userBPage.Goto(gameURL)
		require.NoError(t, err)

		userBPage.WaitForURL("**/select-emoji")

		// User B selects second emoji
		err = userBPage.Click(".emoji-option:nth-child(2)")
		require.NoError(t, err)

		// Both users should enter game simultaneously
		userBPage.WaitForURL("**/game/**")
		userAPage.WaitForURL("**/game/**")

		// Verify both users see same game
		userATitle, err := userAPage.Locator("h2").TextContent()
		require.NoError(t, err)
		userBTitle, err := userBPage.Locator("h2").TextContent()
		require.NoError(t, err)

		assert.Equal(t, userATitle, userBTitle)
		assert.Contains(t, userATitle, fmt.Sprintf("Game #%s", gameID))
		t.Logf("Both users see game title: %s", userATitle)

		// Verify both users see empty game board
		userACells, err := userAPage.Locator(".game-cell").All()
		require.NoError(t, err)
		userBCells, err := userBPage.Locator(".game-cell").All()
		require.NoError(t, err)

		assert.Len(t, userACells, 9, "Should have 9 cells")
		assert.Len(t, userBCells, 9, "Should have 9 cells")

		// Step 3: User A makes first move (top-left corner)
		t.Log("User A: Making first move (top-left)...")
		err = userAPage.Locator(".game-cell").First().Click()
		require.NoError(t, err)

		// Wait for HTMX to update DOM for User A
		_, err = userAPage.WaitForFunction(`document.querySelector('.game-cell').textContent === '🐱'`, nil)
		require.NoError(t, err)

		// Verify User A sees their move
		userAFirstCell, err := userAPage.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🐱", userAFirstCell)
		t.Log("User A: Successfully placed emoji in first cell")

		// Step 4: CRITICAL TEST - Verify User B sees User A's move without refresh
		// This test will FAIL because real-time sync is not implemented
		t.Log("User B: Checking if User A's move is visible (THIS SHOULD FAIL)...")

		// Give some time for potential real-time sync (there isn't any)
		time.Sleep(2 * time.Second)

		userBFirstCell, err := userBPage.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)

		// This assertion was expected to FAIL, but now should PASS due to SSE implementation
		t.Logf("User B first cell content after User A's move: '%s'", userBFirstCell)
		assert.Equal(t, "🐱", userBFirstCell, "User B should see User A's emoji immediately (REAL-TIME SYNC TEST)")

		// Step 5: Demonstrate that User B sees the move after page refresh
		t.Log("User B: Refreshing page to see User A's move...")
		_, err = userBPage.Reload()
		require.NoError(t, err)

		userBFirstCellAfterRefresh, err := userBPage.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🐱", userBFirstCellAfterRefresh, "User B should see User A's emoji after refresh")
		t.Log("User B: Can see User A's emoji after page refresh")

		// Step 6: User B makes second move (center cell)
		t.Log("User B: Making second move (center)...")
		err = userBPage.Locator(".game-cell").Nth(4).Click() // Center cell
		require.NoError(t, err)

		// Wait for HTMX to update DOM for User B
		_, err = userBPage.WaitForFunction(`document.querySelectorAll('.game-cell:not(:empty)').length === 2`, nil)
		require.NoError(t, err)

		// Verify User B sees both moves with different emojis
		userBCatCells, err := userBPage.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🐱",
		}).Count()
		require.NoError(t, err)
		userBRocketCells, err := userBPage.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🚀",
		}).Count()
		require.NoError(t, err)
		assert.Equal(t, 1, userBCatCells, "Should see one 🐱 cell")
		assert.Equal(t, 1, userBRocketCells, "Should see one 🚀 cell")
		t.Log("User B: Successfully placed emoji in center cell")

		// Step 7: Test if User A sees User B's move without refresh (will also fail)
		t.Log("User A: Checking if User B's move is visible (THIS SHOULD ALSO FAIL)...")
		time.Sleep(2 * time.Second)

		userACatCells, err := userAPage.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🐱",
		}).Count()
		require.NoError(t, err)
		userARocketCells, err := userAPage.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🚀",
		}).Count()
		require.NoError(t, err)

		// This was expected to FAIL, but now should PASS due to SSE implementation
		t.Logf("User A sees %d cat cells and %d rocket cells after User B's move", userACatCells, userARocketCells)
		assert.Equal(t, 1, userACatCells, "User A should see one 🐱 cell (REAL-TIME SYNC TEST)")
		assert.Equal(t, 1, userARocketCells, "User A should see one 🚀 cell (REAL-TIME SYNC TEST)")

		// Step 8: Demonstrate eventual consistency after refresh
		t.Log("User A: Refreshing to see User B's move...")
		_, err = userAPage.Reload()
		require.NoError(t, err)

		userACatCellsAfterRefresh, err := userAPage.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🐱",
		}).Count()
		require.NoError(t, err)
		userARocketCellsAfterRefresh, err := userAPage.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🚀",
		}).Count()
		require.NoError(t, err)
		assert.Equal(t, 1, userACatCellsAfterRefresh, "User A should see 🐱 cell after refresh")
		assert.Equal(t, 1, userARocketCellsAfterRefresh, "User A should see 🚀 cell after refresh")

		// Step 9: Test game reset functionality across users
		t.Log("Testing reset functionality across users...")
		err = userAPage.Click("button:text('Reset Game')")
		require.NoError(t, err)

		// User A should see empty board immediately
		_, err = userAPage.WaitForFunction(`document.querySelectorAll('.game-cell:not(:empty)').length === 0`, nil)
		require.NoError(t, err)

		// User B should see reset after refresh (no real-time sync)
		_, err = userBPage.Reload()
		require.NoError(t, err)

		nonEmptyUserBCellsAfterReset, err := userBPage.Locator(".game-cell:not(:empty)").Count()
		require.NoError(t, err)
		assert.Equal(t, 0, nonEmptyUserBCellsAfterReset, "User B should see empty board after reset and refresh")

		t.Log("Test completed - demonstrated current limitations and eventual consistency")
	})
}

func TestMultipleGamesIsolation(t *testing.T) {
	t.Skip("Skipping multiple games isolation test")
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

	// Create contexts for two separate games
	game1Context, err := browser.NewContext()
	require.NoError(t, err)
	defer game1Context.Close()

	game2Context, err := browser.NewContext()
	require.NoError(t, err)
	defer game2Context.Close()

	game1Page, err := game1Context.NewPage()
	require.NoError(t, err)

	game2Page, err := game2Context.NewPage()
	require.NoError(t, err)

	t.Run("Multiple games are isolated from each other", func(t *testing.T) {
		// Create Game 1 with emoji selection
		t.Log("Creating Game 1...")
		_, err = game1Page.Goto(server.URL)
		require.NoError(t, err)

		err = game1Page.Click("a:text('New Game')")
		require.NoError(t, err)

		game1Page.WaitForURL("**/select-emoji")
		err = game1Page.Click(".emoji-option:nth-child(1)") // 🐱
		require.NoError(t, err)

		// Game 1 will be in waiting state since no second player
		game1Page.WaitForSelector(".waiting-state")
		game1URL, err := game1Page.Locator(".url-input").GetAttribute("value")
		require.NoError(t, err)
		game1ID := extractGameID(game1URL)

		// Create Game 2 with emoji selection
		t.Log("Creating Game 2...")
		_, err = game2Page.Goto(server.URL)
		require.NoError(t, err)

		err = game2Page.Click("a:text('New Game')")
		require.NoError(t, err)

		game2Page.WaitForURL("**/select-emoji")
		err = game2Page.Click(".emoji-option:nth-child(2)") // 🚀
		require.NoError(t, err)

		// Game 2 will be in waiting state since no second player
		game2Page.WaitForSelector(".waiting-state")
		game2URL, err := game2Page.Locator(".url-input").GetAttribute("value")
		require.NoError(t, err)
		game2ID := extractGameID(game2URL)

		// Verify different game IDs
		assert.NotEqual(t, game1ID, game2ID, "Games should have different IDs")
		t.Logf("Game 1 ID: %s, Game 2 ID: %s", game1ID, game2ID)

		// Add second players to each game to make them playable
		game1Player2Context, err := browser.NewContext()
		require.NoError(t, err)
		defer game1Player2Context.Close()

		game2Player2Context, err := browser.NewContext()
		require.NoError(t, err)
		defer game2Player2Context.Close()

		game1Player2Page, err := game1Player2Context.NewPage()
		require.NoError(t, err)

		game2Player2Page, err := game2Player2Context.NewPage()
		require.NoError(t, err)

		// Player 2 joins Game 1
		t.Log("Adding second player to Game 1...")
		_, err = game1Player2Page.Goto(game1URL)
		require.NoError(t, err)
		game1Player2Page.WaitForURL("**/select-emoji")
		err = game1Player2Page.Click(".emoji-option:nth-child(3)") // 🎨
		require.NoError(t, err)

		// Both should enter Game 1
		game1Page.WaitForURL("**/game/**")
		game1Player2Page.WaitForURL("**/game/**")

		// Player 2 joins Game 2
		t.Log("Adding second player to Game 2...")
		_, err = game2Player2Page.Goto(game2URL)
		require.NoError(t, err)
		game2Player2Page.WaitForURL("**/select-emoji")
		err = game2Player2Page.Click(".emoji-option:nth-child(4)") // 🌟
		require.NoError(t, err)

		// Both should enter Game 2
		game2Page.WaitForURL("**/game/**")
		game2Player2Page.WaitForURL("**/game/**")

		// Make moves in Game 1
		t.Log("Making moves in Game 1...")
		err = game1Page.Locator(".game-cell").First().Click()
		require.NoError(t, err)
		_, err = game1Page.WaitForFunction(`document.querySelector('.game-cell').textContent === '🐱'`, nil)
		require.NoError(t, err)

		err = game1Page.Locator(".game-cell").Nth(4).Click()
		require.NoError(t, err)
		_, err = game1Page.WaitForFunction(`document.querySelectorAll('.game-cell:not(:empty)').length === 2`, nil)
		require.NoError(t, err)

		// Make different moves in Game 2
		t.Log("Making moves in Game 2...")
		err = game2Page.Locator(".game-cell").Nth(2).Click() // Top-right
		require.NoError(t, err)
		_, err = game2Page.WaitForFunction(`document.querySelector('.game-cell:nth-child(3)').textContent === '🚀'`, nil)
		require.NoError(t, err)

		err = game2Page.Locator(".game-cell").Nth(6).Click() // Bottom-left
		require.NoError(t, err)
		_, err = game2Page.WaitForFunction(`document.querySelectorAll('.game-cell:not(:empty)').length === 2`, nil)
		require.NoError(t, err)

		// Verify Game 1 state
		game1CatCount, err := game1Page.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🐱",
		}).Count()
		require.NoError(t, err)
		assert.Equal(t, 2, game1CatCount)

		game1FirstCell, err := game1Page.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🐱", game1FirstCell, "Game 1 first cell should be 🐱")

		// Verify Game 2 state
		game2RocketCount, err := game2Page.Locator(".game-cell").Filter(playwright.LocatorFilterOptions{
			HasText: "🚀",
		}).Count()
		require.NoError(t, err)
		assert.Equal(t, 2, game2RocketCount)

		game2FirstCell, err := game2Page.Locator(".game-cell").First().TextContent()
		require.NoError(t, err)
		assert.Equal(t, "", game2FirstCell, "Game 2 first cell should be empty")

		game2ThirdCell, err := game2Page.Locator(".game-cell").Nth(2).TextContent()
		require.NoError(t, err)
		assert.Equal(t, "🚀", game2ThirdCell, "Game 2 third cell should be 🚀")

		t.Log("Verified that games are properly isolated from each other")
	})
}
