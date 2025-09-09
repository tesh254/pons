# pons

Turn any public documentation link or local Markdown file into a powerful knowledge base for your local AI tool. Fast, fun, and effortless—just point and connect.

## Features

*   **Web Scraping**: Ingest content from public documentation websites.
*   **Local File Ingestion**: Directly add content from local Markdown files.
*   **Vector Embeddings**: Automatically generate and store vector embeddings for efficient semantic search.
*   **Context Management**: Organize and search documents within specific contexts (e.g., by API, project, or topic).
*   **Model Context Protocol (MCP) Server**: Expose your knowledge base as an MCP server for seamless integration with AI tools.
*   **SQLite Backend**: Reliable and portable local data storage.

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap tesh254/pons
brew install pons
```

### Install Script (Linux and macOS)

```bash
curl -sL https://raw.githubusercontent.com/tesh254/pons/main/install.sh | bash
```

### From Source

(Requires Go installed)

```bash
git clone https://github.com/tesh254/pons.git
cd pons
go build -o pons .
```

## CLI Usage

Pons provides a command-line interface for managing your knowledge base.

### `pons add`

Ingest content from a URL or a local file. This command scrapes web pages or reads local files, generates embeddings, and stores the content in your knowledge base.

```bash
# Add content from a URL (web scraping)
pons add https://www.example.com --context my-web-docs

# Add content from a local Markdown file
pons add /path/to/your/document.md --context my-local-notes

# Use --verbose for detailed output
pons add https://wchr.xyz --context wchr-context --verbose
```

**Arguments:**

*   `[url_or_file_path]`: The URL of the website to scrape or the absolute path to the local Markdown file.

**Flags:**

*   `--context (-c)`: A string to categorize the ingested documents (e.g., `shopify-admin`, `my-project-docs`). Defaults to `default`.
*   `--verbose (-v)`: Enable verbose output for detailed progress and information.

Documents are stored with a `source_type` indicating their origin (`web_scrape` or `file_read`).

### `pons search`

Search your knowledge base for relevant documents using a natural language query.

```bash
# Search across all contexts
pons search "How do I update user profiles?"

# Search within a specific context
pons search "What is the main function?" --context my-project-docs

# Get more results
pons search "Pons features" --num-results 10
```

**Arguments:**

*   `[query]`: The natural language query to search for.

**Flags:**

*   `--context (-c)`: (Optional) The context to search within. If omitted, searches across all contexts.
*   `--num-results (-n)`: The maximum number of search results to return. Defaults to `5`.
*   `--verbose (-v)`: Enable verbose output.

### `pons list`

List all documents currently stored in your knowledge base.

```bash
pons list
```

This command will display the URL, source type, checksum, content length, and embeddings length for each document.

## Using the Pons Model Context Protocol (MCP) Server

The Pons MCP server allows your local AI tools to connect and utilize its capabilities as a knowledge base.

### Starting the Server

To start the server, use the `pons start` command:

```bash
pons start
```

By default, the server listens on `http://localhost:9014`. You can specify a different address and port using the `--http-address` flag:

```bash
pons start --http-address "0.0.0.0:8081"
```

### Connecting Your AI Tool

To connect your AI tool to the Pons MCP server, configure your tool to use the server's address. For example, if your AI tool supports connecting to an MCP server, you would typically provide the `http://localhost:8080` (or your custom address) as the server endpoint.

Refer to your AI tool's documentation for specific instructions on how to configure an MCP server connection.

#### Connecting with Gemini

To connect Gemini to your local Pons MCP server, start Pons with the desired HTTP address:

```bash
pons start --http-address localhost:9014
```

Then, create a folder named `.gemini` in your project's root directory and add a `settings.json` file inside it with the following content:

```json
{
  "mcpServers": {
    "pons": {
      "httpUrl": "http://localhost:9014"
    }
  }
}
```

#### Connecting with Cursor Editor

To connect Cursor Editor to your local Pons MCP server, start Pons with the desired HTTP address:

```bash
pons start --http-address localhost:9999 # Or any other available port
```

Then, create a `.cursor` folder in your project's root directory and add an `mcp.json` file inside it with the following content:

```json
{
    "mcpServers": {
        "pons": {
            "type": "streamable-http",
            "url": "http://localhost:9999",
            "note": "For Streamable HTTP connections, add this URL directly in your MCP Client"
        }
    }
}
```

### MCP Tools Reference

Pons exposes the following MCP tools for AI tool interaction:

#### `learn_api`

🚨 MANDATORY FIRST STEP: This tool MUST be called before any other Pons tools.

⚠️ ALL OTHER PONS TOOLS WILL FAIL without a `conversationId` from this tool.
This tool generates a `conversationId` that is REQUIRED for all subsequent tool calls. After calling this tool, you MUST extract the `conversationId` from the response and pass it to every other Pons tool call.

🔄 MULTIPLE CONTEXT SUPPORT: You MUST call this tool multiple times in the same conversation when you need to learn about different documentation contexts. THIS IS NOT OPTIONAL. Just pass the existing `conversationId` to maintain conversation continuity while loading the new context.

For example, a user might ask a question about the `admin` context, then switch to the `functions` context, then ask a question about `polaris` UI components. In this case, you would call `learn_api` three times with the following arguments:

- `learn_api(api: "admin") -> conversationId: "admin"`
- `learn_api(api: "functions", conversationId: "admin") -> conversationId: "functions"`
- `learn_api(api: "polaris", conversationId: "functions") -> conversationId: "polaris"`

This is because the `conversationId` is used to maintain conversation continuity while loading the new context.

🚨 Valid arguments for `api` are:
    - Any string representing a documentation context (e.g., `shopify-admin`, `my-project-docs`, `general-knowledge`). This string will be used as the `conversationId` for subsequent tool calls.

🔄 WORKFLOW:
1. Call `learn_api` first with the initial API (context)
2. Extract the `conversationId` from the response
3. Pass that same `conversationId` to ALL other Pons tools
4. If you need to know more about a different context at any point in the conversation, call `learn_api` again with the new API (context) and the same `conversationId`

DON'T SEARCH THE WEB WHEN REFERENCING INFORMATION FROM THIS KNOWLEDGE BASE. IT WILL NOT BE ACCURATE.
PREFER THE USE OF THE `search_dataset` TOOL TO RETRIEVE INFORMATION FROM THE KNOWLEDGE BASE.

#### `search_dataset`

Searches the knowledge base for relevant documentation and code examples based on a query string. This tool uses vector embeddings for semantic search.

#### `upsert_document`

Adds or updates a document in the knowledge base, automatically generating embeddings. This tool is used internally by the `pons add` CLI command.

#### `delete_document`

Deletes documents from the knowledge base by URL prefix.

#### `list_documents`

Lists stored documents in the knowledge base with pagination, optionally filtered by context.

#### `get_document`

Retrieves a specific document from the knowledge base by URL.

## Database Backend

Pons uses SQLite (`github.com/mattn/go-sqlite3`) for local data storage. While efforts were made to integrate `libsql` for its native vector capabilities, challenges with its Go driver's compatibility led to reverting to the stable SQLite implementation. Future enhancements may explore more robust vector database integrations.
