# Claude Instructions

## On Startup
- Always read docs/PROJECT_NOTES.md first to understand the project context and current state.
- Always keep project notes updated with any changes or decisions made.
- Always follow Effective Go coding standards and best practices.
- Always use stretchr suite for unit testing.

## Project Commands
- Lint: `make lint`
- Test: `make test`
- Build: `make build`
- Goimports: `find ./ -type f -iname "*.go" -exec goimports -w {} \;`

## Important Files
- docs/PROJECT_NOTES.md - Contains project overview, goals, and current implementation status
- Makefile - Contains build, test, and lint commands
