# Tic-Tac-Toe Multi-User Web Application

## Project Overview & Context

This is a real-time multi-user tic-tac-toe web application built with Go and HTMX. The project demonstrates server-side rendering with real-time synchronization using Server-Sent Events (SSE). Players can create games, share URLs, select emoji identities, and play together with instant move synchronization.

**Tech Stack:**
- Backend: Go with Gin framework
- Frontend: HTMX with SSE extension for real-time updates
- Templates: Go html/template
- Testing: Playwright-Go for E2E tests
- Real-time: Server-Sent Events (SSE)

## Development Principles & Workflow

### Planning-First Approach
- **Always create implementation plans before coding**
- Break complex features into phases (e.g., Phase 1: SSE, Phase 2: Game logic)
- Use TodoWrite tool to track progress and maintain transparency
- Present plans to user for approval before implementation

### Requirement Clarification
- **Ask clarifying questions when requirements are unclear**
- Better to ask 5 questions upfront than implement the wrong feature
- Confirm technical approaches before major implementation work
- Validate assumptions with the user early and often

### High-Level Test-Driven Development
- **Use E2E tests to drive development, not just unit tests**
- Write failing E2E tests first to capture the full user experience
- Tests should cover real browser interactions and multi-user scenarios
- Focus on integration testing over isolated unit tests for web applications

### Iterative Development
- Work in small, testable increments
- Get feedback after each major milestone
- Be prepared to adjust approach based on test results
- Complete one phase fully before moving to the next

## Testing Strategy & Quality Assurance

### E2E Testing with Playwright
- All new features must have E2E test coverage
- Tests should simulate real user workflows
- Use headless mode for CI, non-headless for debugging
- Test multi-user scenarios with separate browser contexts

### Test Execution Commands
```bash
# Run all tests
go test -v ./tests/e2e

# Run specific test
go test -v -run TestEmojiSelection ./tests/e2e

# Run with timeout for long tests
go test -v ./tests/e2e -timeout=60s
```

### Quality Standards
- All tests must pass before considering features complete
- Build must succeed without errors or warnings
- Real-time features must be verified with actual browser testing
- No feature is complete until E2E tests demonstrate it works

## Technical Requirements & Constraints

### Real-Time Architecture
- Use Server-Sent Events (SSE) for real-time updates
- HTMX SSE extension for client-side event handling
- Event broadcasting with subscriber management
- Thread-safe game state management with sync.RWMutex

### Game State Management
- In-memory storage with concurrent access protection
- Player session management via cookies
- Game isolation - no cross-game interference
- State persistence across page refreshes

### Development Commands
```bash
# Build and run
go build -o main . && ./main

# Development server
go run main.go

# Quick test build
go build -o main .
```

## Code Standards & Conventions

### Go Code Style
- Use standard Go formatting (gofmt)
- Implement proper error handling
- Use descriptive variable names
- Keep handlers focused and testable

### Template Organization
- Templates in `templates/layouts/`
- Separate templates for different page types
- Include necessary HTMX and SSE scripts
- Responsive CSS with clean styling

### SSE Event Patterns
```go
// Event broadcasting pattern
broadcastGameEvent(gameID, GameEvent{
    Type:   "move",
    GameID: gameID,
    Data:   gameData,
})

// Thread-safe state updates
gamesMux.Lock()
// modify game state
gamesMux.Unlock()
```

## Architecture & Design Patterns

### Request Flow
1. User creates game → Redirect to emoji selection
2. Emoji selection → Player registered → Redirect to game board
3. Game board loads → SSE connection established
4. User makes move → Event broadcast → Real-time UI update

### Event-Driven Updates
- Use SSE for server-to-client communication
- HTMX handles DOM updates automatically
- Event types: move, reset, player_join, initial
- Client-side event handlers for each event type

### Multi-User Synchronization
- Subscriber pattern for event distribution
- Automatic cleanup on client disconnect
- Context-based cancellation for cleanup
- Per-game event isolation

## Common Workflows

### Adding New Features
1. **Plan**: Create implementation plan with phases
2. **Test**: Write failing E2E tests first
3. **Implement**: Build incrementally with TodoWrite tracking
4. **Verify**: Ensure all tests pass
5. **Iterate**: Refine based on feedback

### Debugging Real-Time Issues
1. Check server logs for SSE connection establishment
2. Verify event broadcasting is working
3. Use browser dev tools to monitor SSE events
4. Test with multiple browser contexts
5. Check for race conditions in concurrent access

### Testing Multi-User Scenarios
```go
// Pattern for multi-user tests
userAContext, err := browser.NewContext()
userBContext, err := browser.NewContext()

userAPage, err := userAContext.NewPage()
userBPage, err := userBContext.NewPage()
```

## Troubleshooting & Common Issues

### Build Failures
- Check for unused variables (Go is strict)
- Verify all imports are used
- Ensure proper variable declarations (`:=` vs `=`)

### Test Failures
- SSE connections take time - add appropriate waits
- Use `WaitForFunction` for dynamic content
- Browser contexts provide session isolation
- Template paths must be correct in test router

### Real-Time Sync Issues
- Verify SSE endpoint is included in router
- Check event broadcasting logic
- Ensure proper context handling for cleanup
- Monitor server logs for connection establishment

## Memory & Context Management

This CLAUDE.md file should be updated iteratively as the project evolves. Use the `#` shortcut during development to quickly add new patterns or insights that work well for this project.

Key reminders:
- Always ask questions before implementing
- Use TodoWrite for transparency
- E2E tests drive development
- Real-time features need browser-based verification