Okay, here is the full Product Requirements Document for **Project-Memory**, incorporating the technical details and structured for easy copy-pasting.

---

````markdown
# Product Requirements Document: Project-Memory (Context Management MCP Tool)

**Version:** 1.1
**Date:** May 12, 2025
**Author:** [Your Name/Team Name]

---

## 1. Executive Summary

This document specifies the technical design and requirements for **Project-Memory**, an independent MCP (Model Context Protocol) Server implementation. The server will function as a persistent, external memory store for MCP Host applications (e.g., AI code editors) to augment LLM context windows. It addresses the inherent limitation of fixed LLM context sizes by providing a mechanism to externalize, process, and retrieve relevant historical interaction context and project-specific details. Project-Memory will expose two primary MCP Tools: `save_context` for externalizing and processing salient context chunks (via summarization and vector embedding) and `retrieve_context` for performing relevance-based vector similarity search over the stored memory. Persistence will be achieved using a local, file-based SQLite database managed by the `crawshaw/sqlite` pure Go driver, ensuring a self-contained, Cgo-free binary suitable for per-project deployment. The project leverages the `github.com/localrivet/gomcp` library for robust MCP protocol implementation.

## 2. Technical Goals

- **G.1:** Implement a fully compliant MCP Server (supporting MCP Spec 2025-03-26 and 2024-11-05) using the `github.com/localrivet/gomcp` library.
- **G.2:** Provide a reliable and durable mechanism for persistent storage of contextual data (text summaries and vector embeddings) utilizing `crawshaw/sqlite` in a local file (`.ctx-memory.db`), adhering to the "100% Go, no C bindings" constraint.
- **G.3:** Expose two primary MCP Tools (`save_context`, `retrieve_context`) that encapsulate the technical workflow for context externalization (summarize, embed, store) and retrieval (query embed, search, return relevant results).
- **G.4:** Ensure the Project-Memory server binary is self-contained, lightweight, and easily deployable on major platforms without external dependencies beyond the generated `.ctx-memory.db` file.
- **G.5:** Design the core components (ContextStore, Summarizer, Embedder) using Go interfaces to promote modularity, testability, and allow for alternative implementations (e.g., different summarization models, embedding providers, or storage backends) in future iterations.

## 3. Architecture and Component Breakdown

The Project-Memory system operates within a client-server architecture where the MCP Host is the client interacting with the Project-Memory MCP Server.

```mermaid
graph TD
    A[MCP Host<br>(e.g., Cursor AI)] --> B{MCP Protocol<br>(via gomcp)}
    B --> C[Project-Memory<br>MCP Server Application]

    C --> D[ContextToolServer Logic]
    D --> E[Summarizer Interface]
    D --> F[Embedder Interface]
    D --> G[ContextStore Interface]

    G --> H[SQLiteContextStore Implementation<br>(using crawshaw/sqlite)]
    H --> I[Persistent Storage<br>(.ctx-memory.db file)]

    E --> E_impl[Summarizer Implementation<br>(Injected)]
    F --> F_impl[Embedder Implementation<br>(Injected)]

    subgraph Project-Memory MCP Server
    C
    D
    E
    F
    G
    H
    I
    end
```
````

**Component Responsibilities:**

- **MCP Host (External):** Initiates MCP tool calls (`save_context`, `retrieve_context`) based on user input or potentially internal logic. Provides input parameters (e.g., `context_text`, `query`). Receives and utilizes structured results from tool calls (e.g., retrieved summaries).
- **`gomcp` Framework:** Handles the low-level MCP protocol details:
  - Manages the server lifecycle and configured transports (e.g., listening on Stdio).
  - Performs protocol negotiation with the Host.
  - Deserializes incoming JSON-RPC messages into tool call requests.
  - Dispatches tool calls to the registered handler functions within `ContextToolServer`.
  - Serializes return values/errors from handlers into JSON-RPC responses and sends them back to the Host.
- **`ContextToolServer` Logic:** Coordinates the application-specific workflow for context management:
  - Implements the `gomcp` server interface (or integrates with it).
  - Registers the `save_context` and `retrieve_context` tools with `gomcp`.
  - Implements the `handleSaveContext` and `handleRetrieveContext` functions, which contain the core logic for processing tool calls.
  - Acts as the orchestrator, calling methods on the `Summarizer`, `Embedder`, and `ContextStore` dependencies.
- **Summarizer Module (`Summarizer` interface):** Abstractly represents the summarization capability. Concrete implementations take raw text or tokens and return a condensed string.
- **Embedder Module (`Embedder` interface):** Abstractly represents the embedding capability. Concrete implementations take text and return its vector representation (`[]float32`).
- **ContextStore Module (`ContextStore` interface):** Abstractly represents the persistent storage interface. Defines methods for storing and searching context entries.
- **`SQLiteContextStore` Implementation:** The concrete implementation of `ContextStore` using `crawshaw/sqlite`. Manages the database connection, executes SQL commands, and handles the specifics of data serialization/deserialization for storage (especially `[]float32` to/from BLOB).
- **Persistent Storage (`.ctx-memory.db`):** The physical SQLite database file located in the project directory, holding structured context data.

**Interaction Flow Details:**

- **`save_context` Workflow:**
  1.  Host sends MCP `CallTool` message for `save_context` with `{ "context_text": "..." }`.
  2.  `gomcp` receives, deserializes, and dispatches to `handleSaveContext`.
  3.  `handleSaveContext` calls `Summarizer.Summarize(...)`.
  4.  `handleSaveContext` calls `Embedder.CreateEmbedding(summary)`.
  5.  `handleSaveContext` generates a unique ID.
  6.  `handleSaveContext` calls `vector.Float32SliceToBytes(embedding)` to serialize.
  7.  `handleSaveContext` calls `ContextStore.Store(id, summary, embedding_bytes, timestamp)`.
  8.  `SQLiteContextStore.Store` prepares and executes `INSERT OR REPLACE INTO context_memory ...` SQL statement via `crawshaw/sqlite`.
  9.  `handleSaveContext` returns a success response `{ "status": "success", "id": "..." }` via `gomcp`.
- **`retrieve_context` Workflow:**
  1.  Host sends MCP `CallTool` message for `retrieve_context` with `{ "query": "..." }`.
  2.  `gomcp` receives, deserializes, and dispatches to `handleRetrieveContext`.
  3.  `handleRetrieveContext` calls `Embedder.CreateEmbedding(query_text)`.
  4.  `handleRetrieveContext` calls `ContextStore.Search(query_embedding, limit)`.
  5.  `SQLiteContextStore.Search` executes `SELECT summary_text, embedding FROM context_memory`.
  6.  `SQLiteContextStore.Search` iterates results: calls `vector.BytesToFloat32Slice(embedding_bytes)`, calculates `vector.CosineSimilarity(query_embedding, stored_embedding)`.
  7.  `SQLiteContextStore.Search` sorts results by similarity and selects top `limit` `summary_text` values.
  8.  `handleRetrieveContext` returns a response `{ "status": "success", "results": [...] }` via `gomcp`.

## 5. Data Model (SQLite Schema)

The persistent context data is stored in a single SQLite table:

```sql
CREATE TABLE IF NOT EXISTS context_memory (
    id TEXT PRIMARY KEY,         -- Unique identifier (e.g., UUID or Content Hash)
    summary_text TEXT NOT NULL,  -- The concise summary of the context chunk
    embedding BLOB NOT NULL,     -- The vector embedding (serialized []float32)
    timestamp INTEGER NOT NULL   -- Unix timestamp of when the context was saved
    -- Future Extension: source_info TEXT -- e.g., JSON string for file path, chat session ID
);
```

**Technical Details:**

- `id`: Chosen as `TEXT PRIMARY KEY` to support GUIDs or content hashes, ensuring uniqueness and efficient lookups if needed directly.
- `summary_text`: Stored as `TEXT`. `NOT NULL` enforced.
- `embedding`: Stored as `BLOB`. This requires serializing the `[]float32` slice into a byte array before insertion and deserializing it after selection. `NOT NULL` enforced.
- `timestamp`: Stored as an `INTEGER` representing a Unix timestamp (`time.Now().Unix()`). Useful for sorting by recency or future cleanup operations. `NOT NULL` enforced.

## 6. Functional Requirements (Technical Implementation)

- **FR.6.1:** The `main` package shall initialize the `gomcp` server, instantiate dependencies (`Summarizer`, `Embedder`), initialize the `SQLiteContextStore` (providing the `.ctx-memory.db` path), inject dependencies into the `ContextToolServer`, and start the `gomcp` server listener.
- **FR.6.2:** The `contextstore` package shall contain the `ContextStore` interface and the `SQLiteContextStore` implementation using `golang.org/x/exp/sqlite.org/v1`.
- **FR.6.2.1:** `SQLiteContextStore.Init` shall handle the `sqlite3.Open` call and the `CREATE TABLE IF NOT EXISTS` statement. Error handling must include proper closing of the connection on failure.
- **FR.6.2.2:** `SQLiteContextStore.Close` shall call the `sqlite3.Conn.Close` method.
- **FR.6.2.3:** `SQLiteContextStore.Store` shall prepare an `INSERT OR REPLACE` statement, serialize the `[]float32` embedding using `vector.Float32SliceToBytes`, bind parameters using `sqlite3.Bind`, and execute via `stmt.Step()`.
- **FR.6.2.4:** `SQLiteContextStore.Search` shall prepare a `SELECT id, summary_text, embedding FROM context_memory` statement. It shall iterate results using `stmt.Step()`, bind columns using `stmt.Scan`, deserialize the `BLOB` embedding using `vector.BytesToFloat32Slice`. It shall then calculate cosine similarity between the `queryEmbedding` and _each_ retrieved embedding in Go (`vector.CosineSimilarity`), sort the results by similarity (descending), and return the `summary_text` of the top `limit` entries.
- **FR.6.3:** The `summarizer` package shall contain the `Summarizer` interface and a concrete implementation.
- **FR.6.4:** The `vector` package shall contain the `Embedder` interface, a concrete implementation, and helper functions `Float32SliceToBytes`, `BytesToFloat32Slice`, and `CosineSimilarity([]float32, []float32) float64`.

## 7. Non-Functional Requirements (Technical Constraints & Qualities)

- **NFR.7.1 - Performance (Search):** Search latency is directly dependent on the number of entries (N) and embedding dimension (E) due to the O(N \* E) similarity calculation in Go. Performance for N > ~few thousand entries with typical embedding sizes may become noticeable and will be a target for future optimization.
- **NFR.7.2 - Performance (Store):** Storage latency should be dominated by SQLite write speed on local disk, plus O(E) for serialization.
- **NFR.7.3 - Memory Usage:** Peak memory usage during search will be O(N \* E) for storing all embeddings retrieved from the database before sorting. This should be managed; logging memory usage may be necessary for optimization.
- **NFR.7.4 - Reliability:** Data persistence is guaranteed by SQLite's transactional nature. Database file integrity relies on the robustness of `crawshaw/sqlite` and the underlying filesystem.
- **NFR.7.5 - Portability:** No reliance on system C libraries or external runtime dependencies beyond the Go standard library and specified pure Go modules (`gomcp`, `crawshaw/sqlite`).
- **NFR.7.6 - File Management:** The server must be configured to use a specific `.ctx-memory.db` file path, ideally defaulting to the current working directory or a configurable path relative to it, to support per-project memory.

## 8. Out of Scope (Technical)

- Implementation of the `Summarizer` and `Embedder` interfaces beyond basic mocks. These are external dependencies to this core MCP server and storage project.
- Advanced SQL schema design (e.g., indexing strategies other than primary key, using JOINs).
- Implementing a distributed or shared context memory solution.
- Automatic data synchronization or merging across different instances or machines.
- Providing mechanisms for manual editing or viewing of the `.ctx-memory.db` file contents outside of the defined MCP tools.
- Complex access control or user authentication within the MCP server itself (relying on the Host/Transport layer for any necessary security).

## 9. Future Considerations (Technical)

- Implement more efficient search strategies for large datasets, potentially integrating file-based vector index libraries (e.g., Annoy, FAISS) in Go or exploring SQLite extensions if compatible with `crawshaw/sqlite`.
- Add support for storing and querying multiple types of context entries (e.g., raw messages, key-value facts, code snippets) with appropriate metadata and schema extensions.
- Implement automated database cleanup based on age or other criteria.
- Explore alternative pure Go embedded databases or file formats (e.g., BoltDB, BadgerDB) if SQLite proves limiting for specific access patterns.
- Add instrumentation and logging for performance monitoring and debugging of tool calls and database operations.

## 10. Metrics (Optional - Technical)

- Average and P95 latency for `save_context` tool calls.
- Average and P95 latency for `retrieve_context` tool calls as N (number of entries) increases.
- Database file size over time.
- Number of entries in the `context_memory` table.
- Memory usage profile of the server application.

---

```

```
