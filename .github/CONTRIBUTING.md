# Contributing

Issues and pull requests are very welcome. Please follow conventional commits and run `make lint test` before opening a PR.

Before making any changes to this repository, we kindly request you to initiate discussions for proposed changes that do not yet have an associated [issue](https://github.com/brizzai/auto-mcp/issues). Your collaboration is greatly appreciated.

Please note: we have a [code of conduct](https://github.com/brizzai/auto-mcp/blob/master/.github/CODE_OF_CONDUCT.md), please follow it in all your interactions with the `Auto MCP` project.

---

## Project Structure

Auto MCP follows the standard Go project layout:

- **`cmd/auto-mcp/`**: Contains the main entry point for the application
  - `main.go`: The main function that serves as the entry point
- **`internal/`**: Private application code not meant to be imported by other projects
  - `config/`: Configuration loading and parsing
  - `logger/`: Application logging setup
  - `parser/`: Swagger/OpenAPI parsing
  - `requester/`: Handles external API requests
  - `server/`: MCP server implementation (STDIO/SSE)
- **`build/`**: Compiled application binaries

To build the project:

```bash
make build   # Binary will be in build/auto-mcp
```

---

## Release & Versioning

Auto MCP uses [GoReleaser](https://goreleaser.com/) to automate builds and publish cross-platform releases. For comprehensive information about the release process, automated GitHub Actions workflows, and available artifacts, see [RELEASE.md](RELEASE.md).

Check the current version with:

```bash
auto-mcp --version
```

---

## Pull Requests or Commits
Titles always we must use prefix according to below:

> 🔥 Feature, ♻️ Refactor, 🩹 Fix, 🚨 Test, 📚 Doc, 🎨 Style
- 🔥 Feature: Add SSE transport timeout configuration
- ♻️ Refactor: Rename HTTPRequester to APIRequester
- 🩹 Fix: Improve OpenAPI v3 schema parsing
- 🚨 Test: Validate auth token handling in requests
- 📚 Doc: Add section on custom middleware integration
- 🎨 Style: Apply consistent error handling pattern

All pull requests that contain a feature or fix are mandatory to have unit tests. Your PR is only to be merged if you respect this flow.

## Pre-commit Hooks

This project uses pre-commit hooks to ensure code quality and consistency. Before contributing, please set up pre-commit:

```bash
pip install pre-commit
pre-commit install
```

Pre-commit will automatically run various checks on your code when you commit, ensuring that your contributions meet the project's standards.

---

## Running Example

The repository includes a ready-to-run example using the Swagger [PetStore](http://petstore.swagger.io/v2) API with Auto MCP:

```bash
# Start the service in SSE mode (runs on port 8080 by default)
docker compose -f examples/petshop/docker-compose.yml up
```

Once running, you can access the MCP SSE endpoint at `http://localhost:8080/sse`.

You can inspect and test your newly created MCP using the MCP Inspector:

```bash
npx @modelcontextprotocol/inspector
```

---


# 👍 Contribute

If you want to say **thank you** and/or support the active development of `AutoMCP`:

1. Add a [GitHub Star](https://github.com/brizzai/auto-mcp/stargazers) to the project.
2. Tweet about the project [on your 𝕏 (Twitter)](https://twitter.com/intent/tweet?text=%F0%9F%9A%80%20Auto%20MCP%20instantly%20spins%20your%20OpenAPI%2FSwagger%20spec%20into%20a%20live%20Model%20Context%20Protocol%20server%E2%80%94zero%20boilerplate%2C%20flexible%20auth%2C%20multi-transport%2C%20LLM-ready!%20%23OpenAPI%20%23LLM%20%23MCP&url=https%3A%2F%2Fgithub.com%2Fbrizzai%2Fauto-mcp%20).
3. Write a review or tutorial on [Medium](https://medium.com/), [Dev.to](https://dev.to/) or personal blog.
