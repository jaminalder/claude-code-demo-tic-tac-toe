package e2e

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicTacToeGameLogic(t *testing.T) {
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

	t.Run("Turn alternation - Player 1 starts, then Player 2", func(t *testing.T) {
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

		// Verify Player 1 (🐱) turn indicator is shown
		err = userAPage.Locator(".turn-indicator").WaitFor()
		require.NoError(t, err)

		turnIndicator, err := userAPage.Locator(".turn-indicator").TextContent()
		require.NoError(t, err)

		// Clean up whitespace for comparison
		turnIndicator = strings.TrimSpace(turnIndicator)
		turnIndicator = strings.ReplaceAll(turnIndicator, "\n", " ")
		turnIndicator = strings.ReplaceAll(turnIndicator, "\t", " ")
		for strings.Contains(turnIndicator, "  ") {
			turnIndicator = strings.ReplaceAll(turnIndicator, "  ", " ")
		}

		assert.Contains(t, turnIndicator, "🐱", "Should show Player 1's turn initially")
		assert.Contains(t, strings.ToLower(turnIndicator), "turn", "Should indicate it's their turn")

		// Player 1 makes first move (top-left)
		err = userAPage.Locator(".game-cell").First().Click()
		require.NoError(t, err)

		// Wait for move to process
		_, err = userAPage.WaitForFunction(`document.querySelector('.game-cell').textContent === '🐱'`, nil)
		require.NoError(t, err)

		// Wait for page reload and verify turn indicator now shows Player 2's turn
		time.Sleep(1500 * time.Millisecond) // Allow SSE to trigger page reload
		err = userAPage.Locator(".turn-indicator").WaitFor()
		require.NoError(t, err)

		turnIndicator, err = userAPage.Locator(".turn-indicator").TextContent()
		require.NoError(t, err)

		// Clean up whitespace for comparison
		turnIndicator = strings.TrimSpace(turnIndicator)
		turnIndicator = strings.ReplaceAll(turnIndicator, "\n", " ")
		turnIndicator = strings.ReplaceAll(turnIndicator, "\t", " ")
		for strings.Contains(turnIndicator, "  ") {
			turnIndicator = strings.ReplaceAll(turnIndicator, "  ", " ")
		}

		assert.Contains(t, turnIndicator, "🚀", "Should show Player 2's turn after Player 1 moves")

		// Verify Player 1 cannot make another move immediately
		err = userAPage.Locator(".game-cell").Nth(1).Click()
		require.NoError(t, err)

		// Wait a bit and verify the cell is still empty (move should be rejected)
		time.Sleep(500 * time.Millisecond)
		secondCellContent, err := userAPage.Locator(".game-cell").Nth(1).TextContent()
		require.NoError(t, err)
		assert.Empty(t, secondCellContent, "Player 1 should not be able to move when it's Player 2's turn")

		// Player 2 makes their move (center)
		err = userBPage.Locator(".game-cell").Nth(4).Click()
		require.NoError(t, err)

		// Wait for move to process
		_, err = userBPage.WaitForFunction(`document.querySelectorAll('.game-cell')[4].textContent === '🚀'`, nil)
		require.NoError(t, err)

		// Verify turn indicator shows Player 1's turn again
		time.Sleep(500 * time.Millisecond) // Allow SSE to update
		turnIndicator, err = userBPage.Locator(".turn-indicator").TextContent()
		require.NoError(t, err)
		assert.Contains(t, turnIndicator, "🐱", "Should show Player 1's turn after Player 2 moves")

		t.Logf("Game ID: %s - Turn alternation working correctly", gameID)
	})

	t.Run("Winner detection - Player 1 wins with top row", func(t *testing.T) {
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

		// Play a game where Player 1 wins with top row
		// Player 1: (0,0), Player 2: (1,0), Player 1: (0,1), Player 2: (1,1), Player 1: (0,2) - WINS

		// Move 1: Player 1 (0,0)
		err = userAPage.Locator(".game-cell").Nth(0).Click() // Top-left
		require.NoError(t, err)
		_, err = userAPage.WaitForFunction(`document.querySelectorAll('.game-cell')[0].textContent === '🐱'`, nil)
		require.NoError(t, err)

		// Move 2: Player 2 (1,0)
		time.Sleep(200 * time.Millisecond)
		err = userBPage.Locator(".game-cell").Nth(3).Click() // Middle-left
		require.NoError(t, err)
		_, err = userBPage.WaitForFunction(`document.querySelectorAll('.game-cell')[3].textContent === '🚀'`, nil)
		require.NoError(t, err)

		// Move 3: Player 1 (0,1)
		time.Sleep(200 * time.Millisecond)
		err = userAPage.Locator(".game-cell").Nth(1).Click() // Top-middle
		require.NoError(t, err)
		_, err = userAPage.WaitForFunction(`document.querySelectorAll('.game-cell')[1].textContent === '🐱'`, nil)
		require.NoError(t, err)

		// Move 4: Player 2 (1,1)
		time.Sleep(200 * time.Millisecond)
		err = userBPage.Locator(".game-cell").Nth(4).Click() // Center
		require.NoError(t, err)
		_, err = userBPage.WaitForFunction(`document.querySelectorAll('.game-cell')[4].textContent === '🚀'`, nil)
		require.NoError(t, err)

		// Move 5: Player 1 (0,2) - WINNING MOVE
		time.Sleep(200 * time.Millisecond)
		err = userAPage.Locator(".game-cell").Nth(2).Click() // Top-right
		require.NoError(t, err)
		_, err = userAPage.WaitForFunction(`document.querySelectorAll('.game-cell')[2].textContent === '🐱'`, nil)
		require.NoError(t, err)

		// Wait for winner detection
		time.Sleep(1000 * time.Millisecond)

		// Verify winner announcement is shown
		gameResult, err := userAPage.Locator(".game-result").TextContent()
		require.NoError(t, err)
		assert.Contains(t, gameResult, "🐱", "Winner announcement should show Player 1's emoji")
		assert.Contains(t, gameResult, "wins", "Should announce Player 1 as winner")

		// Verify both players see the same result
		gameResultB, err := userBPage.Locator(".game-result").TextContent()
		require.NoError(t, err)
		assert.Equal(t, gameResult, gameResultB, "Both players should see the same winner announcement")

		// Verify no more moves are allowed
		err = userBPage.Locator(".game-cell").Nth(5).Click() // Try to click bottom-middle
		require.NoError(t, err)

		// Wait and verify the cell is still empty (game should be finished)
		time.Sleep(500 * time.Millisecond)
		cellContent, err := userBPage.Locator(".game-cell").Nth(5).TextContent()
		require.NoError(t, err)
		assert.Empty(t, cellContent, "No moves should be allowed after game is won")

		// Verify turn indicator is hidden or shows game over
		turnIndicatorVisible, err := userAPage.Locator(".turn-indicator").IsVisible()
		if err == nil && turnIndicatorVisible {
			turnText, _ := userAPage.Locator(".turn-indicator").TextContent()
			assert.NotContains(t, turnText, "turn", "Turn indicator should not show active turn when game is over")
		}

		t.Logf("Game ID: %s - Winner detection working correctly", gameID)
	})

	t.Run("Draw detection - full board with no winner", func(t *testing.T) {
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

		// Play a game that results in a draw
		// Board layout that creates a draw:
		// 🐱 🚀 🐱
		// 🚀 🐱 🚀
		// 🚀 🐱 🚀

		moves := []struct {
			player string
			page   playwright.Page
			index  int
		}{
			{"Player1", userAPage, 1}, // (0,0) 🐱
			{"Player2", userBPage, 0}, // (0,1) 🚀
			{"Player1", userAPage, 3}, // (0,2) 🐱
			{"Player2", userBPage, 2}, // (1,0) 🚀
			{"Player1", userAPage, 5}, // (1,1) 🐱
			{"Player2", userBPage, 4}, // (1,2) 🚀
			{"Player1", userAPage, 6}, // (2,1) 🐱
			{"Player2", userBPage, 7}, // (2,0) 🚀
			{"Player1", userAPage, 8}, // (2,2) 🐱 - This should be the final move creating a draw
		}

		// Execute all moves
		for i, move := range moves {
			t.Logf("Move %d: %s clicking cell %d", i+1, move.player, move.index)

			err = move.page.Locator(".game-cell").Nth(move.index).Click()
			require.NoError(t, err)

			// Wait for move to be processed
			time.Sleep(300 * time.Millisecond)

			// Verify the move was placed (except we don't know what the final emoji will be)
			cellContent, err := move.page.Locator(".game-cell").Nth(move.index).TextContent()
			require.NoError(t, err)
			assert.NotEmpty(t, cellContent, "Cell should have emoji after move")
		}

		// Wait for draw detection
		time.Sleep(1000 * time.Millisecond)

		// Verify draw announcement is shown
		gameResult, err := userAPage.Locator(".game-result").TextContent()
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(gameResult), "draw", "Should announce draw result")

		// Verify both players see the same result
		gameResultB, err := userBPage.Locator(".game-result").TextContent()
		require.NoError(t, err)
		assert.Equal(t, gameResult, gameResultB, "Both players should see the same draw announcement")

		t.Logf("Game ID: %s - Draw detection working correctly", gameID)
	})

	t.Run("All winning combinations work correctly", func(t *testing.T) {
		// Test all 8 possible winning combinations:
		// Rows: (0,0),(0,1),(0,2) | (1,0),(1,1),(1,2) | (2,0),(2,1),(2,2)
		// Columns: (0,0),(1,0),(2,0) | (0,1),(1,1),(2,1) | (0,2),(1,2),(2,2)
		// Diagonals: (0,0),(1,1),(2,2) | (0,2),(1,1),(2,0)

		winningCombinations := []struct {
			name  string
			cells []int // indices of winning cells
		}{
			{"Top Row", []int{0, 1, 2}},
			{"Middle Row", []int{3, 4, 5}},
			{"Bottom Row", []int{6, 7, 8}},
			{"Left Column", []int{0, 3, 6}},
			{"Middle Column", []int{1, 4, 7}},
			{"Right Column", []int{2, 5, 8}},
			{"Main Diagonal", []int{0, 4, 8}},
			{"Anti Diagonal", []int{2, 4, 6}},
		}

		for _, combo := range winningCombinations {
			t.Run(combo.name, func(t *testing.T) {
				// Setup fresh game for each test
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

				gameID := setupTwoPlayerGame(t, server.URL, userAPage, userBPage)

				// Execute moves to achieve this winning combination
				// Player 1 will win with this combination, Player 2 plays other cells
				otherCells := []int{}
				for i := 0; i < 9; i++ {
					found := false
					for _, winCell := range combo.cells {
						if i == winCell {
							found = true
							break
						}
					}
					if !found {
						otherCells = append(otherCells, i)
					}
				}

				// Play the game: Player1 takes winning cells, Player2 takes others
				for i := 0; i < 3; i++ {
					// Player 1 move (winning combination)
					err = userAPage.Locator(".game-cell").Nth(combo.cells[i]).Click()
					require.NoError(t, err)
					time.Sleep(200 * time.Millisecond)

					if i < 2 { // Don't let Player 2 move after Player 1 wins
						// Player 2 move (non-winning cell)
						if i < len(otherCells) {
							err = userBPage.Locator(".game-cell").Nth(otherCells[i]).Click()
							require.NoError(t, err)
							time.Sleep(200 * time.Millisecond)
						}
					}
				}

				// Wait for winner detection
				time.Sleep(1000 * time.Millisecond)

				// Verify Player 1 wins
				gameResult, err := userAPage.Locator(".game-result").TextContent()
				require.NoError(t, err)
				assert.Contains(t, gameResult, "🐱", "Player 1 should win with "+combo.name)
				assert.Contains(t, strings.ToLower(gameResult), "win", "Should announce win")

				t.Logf("Game ID: %s - %s winning combination works", gameID, combo.name)
			})
		}
	})
}

// Helper function to setup a complete 2-player game
func setupTwoPlayerGame(t *testing.T, serverURL string, userAPage, userBPage playwright.Page) string {
	// User A creates game and selects emoji
	_, err := userAPage.Goto(serverURL)
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
	_, err = userBPage.Goto(gameURL)
	require.NoError(t, err)

	userBPage.WaitForURL("**/select-emoji")
	err = userBPage.Click(".emoji-option:nth-child(2)") // 🚀
	require.NoError(t, err)

	// Both should enter the game
	userAPage.WaitForURL("**/game/**")
	userBPage.WaitForURL("**/game/**")

	// Extract game ID from URL
	gameURL = userAPage.URL()
	gameID := extractGameID(gameURL)

	time.Sleep(500 * time.Millisecond) // Allow initial sync

	return gameID
}
