// Global JavaScript for Tic-Tac-Toe Application

// Common HTMX configuration
document.body.addEventListener('htmx:configRequest', (event) => {
    event.detail.headers['X-Requested-With'] = 'XMLHttpRequest';
});

// Game ready event handler for emoji selection page
document.addEventListener('htmx:sse-message', function(event) {
    if (event.detail.type === 'game_ready') {
        // Extract game ID from current URL
        const currentPath = window.location.pathname;
        const gameIdMatch = currentPath.match(/\/game\/([^\/]+)\//);
        if (gameIdMatch) {
            const gameId = gameIdMatch[1];
            window.location.href = '/game/' + gameId;
        }
    }
});

// Game events for UI updates (SSE handles most updates automatically)
// Additional game-specific JavaScript can be added here as needed