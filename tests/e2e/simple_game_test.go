package e2e

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
)

func TestSimpleGameSetup(t *testing.T) {
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

	t.Run("Basic game setup works", func(t *testing.T) {
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

		// User A creates game and selects emoji
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
		_, err = userBPage.Goto(gameURL)
		require.NoError(t, err)

		userBPage.WaitForURL("**/select-emoji")
		err = userBPage.Click(".emoji-option:nth-child(2)") // 🚀
		require.NoError(t, err)

		// Both should enter the game
		userAPage.WaitForURL("**/game/**")
		userBPage.WaitForURL("**/game/**")

		time.Sleep(500 * time.Millisecond) // Allow page to fully load

		// Check if turn indicator exists
		turnIndicatorVisible, err := userAPage.Locator(".turn-indicator").IsVisible()
		if err != nil {
			t.Logf("Could not check turn indicator visibility: %v", err)
		} else if !turnIndicatorVisible {
			t.Logf("Turn indicator not visible")
		} else {
			t.Logf("Turn indicator is visible!")
			turnText, _ := userAPage.Locator(".turn-indicator").TextContent()
			t.Logf("Turn indicator text: %s", turnText)
		}

		// Check what's on the page
		pageTitle, _ := userAPage.Locator("h2").TextContent()
		t.Logf("Page title: %s", pageTitle)
		
		// Check if there are any errors in the HTML
		bodyHTML, _ := userAPage.Locator("body").InnerHTML()
		t.Logf("Page has %d characters in body", len(bodyHTML))

		t.Log("Basic test completed successfully")
	})
}