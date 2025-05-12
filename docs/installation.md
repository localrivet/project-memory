# Installation Guide

This guide provides detailed instructions for installing and setting up Project-Memory.

## Prerequisites

Before installing Project-Memory, ensure you have the following prerequisites installed:

- **Go**: Version 1.20 or higher. [Download Go](https://golang.org/dl/)
- **SQLite**: The database engine used for persistent storage. It's usually pre-installed on most systems.

## Installation Methods

### Method 1: Using `go get`

The simplest way to install Project-Memory is using Go's package manager:

```bash
go get github.com/localrivet/projectmemory
```

This will download the source code and its dependencies.

### Method 2: Cloning the Repository

For development or to make modifications, you can clone the repository:

```bash
git clone https://github.com/localrivet/projectmemory.git
cd project-memory
go mod download  # Download dependencies
```

## Configuration

After installation, you need to create a configuration file. Create a `.projectmemoryconfig` file in your project root:

```json
{
  "models": {
    "provider": "mock",
    "modelId": "mock-model",
    "maxTokens": 1024,
    "temperature": 0.7
  },
  "database": {
    "path": ".projectmemory.db"
  },
  "logging": {
    "level": "INFO",
    "format": "TEXT"
  }
}
```

### Configuration Options

For a detailed explanation of all configuration options, see [Configuration Reference](configuration.md).

## Running the Server

Once installed and configured, you can run the Project-Memory server:

```bash
# If installed via go get
go run github.com/localrivet/projectmemory/cmd/project-memory

# If cloned from repository
cd project-memory
go run cmd/project-memory/main.go
```

## Verifying Installation

To verify that Project-Memory is running correctly:

1. The server will output log messages to the console.
2. Look for the message: "MCP server initialized successfully with 2 tools"
3. The server will be listening for MCP tool calls.

## Troubleshooting

### Common Issues

- **Configuration file not found**: Ensure the `.projectmemoryconfig` file is in the directory where you're running the server.
- **Database errors**: Check that the path in your config file is writable by the current user.
- **Port conflicts**: If you're integrating with another MCP server, ensure there are no port conflicts.

### Getting Help

If you encounter issues not covered in this guide:

- Check the [GitHub Issues](https://github.com/localrivet/projectmemory/issues) for similar problems
- Create a new issue with detailed information about your environment and the error
