# Architecture Documentation

This document describes the architecture of Project-Memory, explaining its components, their interactions, and the design decisions behind them.

## System Overview

Project-Memory is designed as a context management system for Large Language Models (LLMs), providing persistent storage and retrieval of relevant information across conversations. It follows a modular architecture with clear separation of concerns.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Project-Memory Server                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐     ┌──────────────┐    ┌───────────────┐  │
│  │             │     │              │    │               │  │
│  │  MCP Server │◄────┤ Context Tool │◄───┤   Tool        │  │
│  │             │     │    Server    │    │ Handlers      │  │
│  └─────────────┘     └──────────────┘    └───────────────┘  │
│         ▲                    ▲                  ▲           │
│         │                    │                  │           │
│         │                    │                  │           │
│         ▼                    ▼                  ▼           │
│  ┌─────────────┐     ┌──────────────┐    ┌───────────────┐  │
│  │             │     │              │    │               │  │
│  │Logging      │     │  Telemetry   │    │Configuration  │  │
│  │System       │     │              │    │Management     │  │
│  └─────────────┘     └──────────────┘    └───────────────┘  │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                  Core Functionality                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐     ┌──────────────┐    ┌───────────────┐  │
│  │             │     │              │    │               │  │
│  │  SQLite     │     │  Summarizer  │    │  Vector       │  │
│  │ContextStore │     │              │    │  Operations   │  │
│  └─────────────┘     └──────────────┘    └───────────────┘  │
│         ▲                    ▲                  ▲           │
│         │                    │                  │           │
│         │                    │                  │           │
│         │                    ▼                  │           │
│         │           ┌──────────────┐            │           │
│         │           │              │            │           │
│         └───────────┤   Database   ├────────────┘           │
│                     │              │                        │
│                     └──────────────┘                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Component Descriptions

### MCP Server

The MCP (Model Context Protocol) server is the communication layer for Project-Memory. It exposes tools that can be invoked by language models or other applications.

**Key Components:**

- **Tool Registration**: Manages the available tools
- **Request Handling**: Processes incoming tool calls
- **Response Formatting**: Formats responses according to the MCP protocol

### Context Tool Server

The Context Tool Server is the main orchestrator, initializing components and registering tool handlers with the MCP server.

**Key Responsibilities:**

- Initialization of core components
- Tool handler registration
- Server lifecycle management

### Core Components

#### ContextStore

The ContextStore handles the storage and retrieval of context information.

**Features:**

- **Initialization**: Sets up the SQLite database
- **Storage**: Saves context entries with metadata and embeddings
- **Search**: Performs semantic search to find relevant context
- **Maintenance**: Handles database maintenance operations

#### Summarizer

The Summarizer component condenses text into meaningful summaries for more efficient storage and retrieval.

**Features:**

- **Provider Management**: Supports different AI providers
- **Fallback Mechanism**: Can fall back to alternative providers if a primary one fails
- **Configuration**: Configurable summary length and parameters

#### Vector Operations

Provides utilities for working with vector embeddings.

**Features:**

- **Embedding Creation**: Generates embeddings from text
- **Similarity Calculation**: Computes cosine similarity between vectors
- **Serialization**: Converts between float arrays and byte representations

### Supporting Components

#### Logger

A structured logging system that provides consistent logging across the application.

**Features:**

- **Log Levels**: Configurable log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- **Structured Data**: Supports adding fields to log entries
- **Formatting Options**: Supports both text and JSON output

#### Configuration Management

Handles loading and parsing configuration from various sources.

**Features:**

- **File-based Configuration**: Loads from .projectmemoryconfig
- **Environment Variables**: Supports overriding settings with environment variables
- **Validation**: Validates configuration values

#### Telemetry

Collects performance metrics for monitoring system behavior.

**Features:**

- **Request Timing**: Measures response times
- **Database Metrics**: Tracks database operations
- **Rate Limiting**: Monitors API usage

## Data Flow

### Save Context Flow

1. Client sends context text to the `save_context` tool
2. Text is summarized by the Summarizer component
3. Summarizer uses AI provider to generate a concise summary
4. Vector embeddings are created for the summary
5. Summary and embeddings are stored in the SQLite database
6. Response with context ID is returned to the client

### Retrieve Context Flow

1. Client sends a query to the `retrieve_context` tool
2. Vector embeddings are created for the query
3. Cosine similarity search is performed against stored embeddings
4. The most similar context entries are retrieved
5. Results are returned to the client

## Design Decisions

### SQLite Storage

SQLite was chosen for several reasons:

- No external database dependencies
- Good performance for the expected workload
- Support for blob storage (for embeddings)
- Cross-platform compatibility

### Interface-Based Design

The system uses interfaces for key components to allow for:

- Easy testing with mock implementations
- Flexibility to swap implementations
- Clear separation of concerns

### Error Handling

A structured approach to error handling:

- Categorized errors (validation, database, network, etc.)
- Consistent error format in responses
- Detailed logging for debugging

### Configuration

A flexible configuration system:

- JSON-based configuration file
- Environment variable fallbacks
- Provider-specific settings

## Performance Considerations

- **Embedding Caching**: Frequently accessed embeddings can be cached
- **Summarization**: Summarizing text reduces storage requirements and improves search performance
- **SQLite Optimizations**: Proper indexing for vector similarity searches
- **Vector Operations**: Efficient implementation of vector operations

## Security Considerations

- **API Key Management**: Secure handling of provider API keys
- **Database Security**: Proper file permissions for the SQLite database
- **Input Validation**: Validation of all input parameters
- **Error Message Security**: Careful exposure of error details to clients
