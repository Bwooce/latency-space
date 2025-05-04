# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Run Commands
- **Go Proxy**: `cd proxy/src && go build` or `go build -o test_socks test_socks.go`
- **Run Tests**: `cd proxy/src && go test -v ./...` or for a single test: `go test -v -run TestName`
- **Status Frontend**: `cd status && npm run dev` (development) or `npm run build` (production)
- **Docker**: `docker compose up -d` (all services) or `docker compose -f docker-compose.minimal.yml up -d` (minimal)
- **Diagnostic Information**: `curl https://latency.space/diagnostic.html` will provide current running instance diagnositic information

## Code Style Guidelines
- **Go**: Standard Go formatting with proper error handling (always check err != nil). Always run tests after making changes.
- **JavaScript**: Use ES6 features, React functional components with hooks
- **Imports**: Group standard library, third-party, and local imports
- **Naming**: Use camelCase for JS/React, snake_case for filenames, PascalCase for Go exports
- **Error Handling**: Log all errors, prefer early returns over nested conditionals
- **Documentation**: Add comments for functions explaining purpose, parameters, and return values
- **Testing**: Write tests for all new functionality, aim for high coverage
- **Security**: Validate all inputs, especially in the proxy components
