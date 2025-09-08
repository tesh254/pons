package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tesh254/pons/internal/api"
	"github.com/tesh254/pons/internal/storage"
)

type Core struct {
}

type Content struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

type SearchDataset struct {
	Query string `json:"query" jsonschema:"required"`
	ConversationID string `json:"conversationId,omitempty"`
}

type UpsertDocumentArgs struct {
	URL         string `json:"url" jsonschema:"required"`
	Content     string `json:"content" jsonschema:"required"`	
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

type DeleteDocumentArgs struct {
	URLPrefix string `json:"url_prefix" jsonschema:"required"`
	ConversationID string `json:"conversationId,omitempty"`
}

type ListDocumentsArgs struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

type GetDocumentArgs struct {
	URL string `json:"url" jsonschema:"required"`
	ConversationID string `json:"conversationId,omitempty"`
}

type LearnApiArgs struct {
	Api            string `json:"api" jsonschema:"required"`
	ConversationID string `json:"conversationId,omitempty"`
}

type SearchDatasetTopKArgs struct {
	Query     string  `json:"query" jsonschema:"required"`
	TopK      int     `json:"top_k" jsonschema:"required"`
	Threshold float64 `json:"threshold,omitempty"`
}

func (c *Core) StartServer(internalAPI *api.API, httpAddress string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "Pons MCP Server", Version: "v1.0.0"}, nil)
	c.registerTools(server, internalAPI)

	if httpAddress != "" {
		return c.ServeHTTP(server, httpAddress)
	}

	return c.ServeStdio(server)
}

func (c *Core) ServeHTTP(server *mcp.Server, httpAddress string) error {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
	log.Printf("Pons MCP handler listening at %s", httpAddress)
	return http.ListenAndServe(httpAddress, loggingHandler(handler))
}

func (c *Core) ServeStdio(server *mcp.Server) error {
	ctx := context.Background()
	transport := &mcp.StdioTransport{}
	t := &mcp.LoggingTransport{Transport: transport, Writer: os.Stderr}
	log.Printf("Starting Pons MCP server with stdio transport")
	return server.Run(ctx, t)
}

type DocumentOutput struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Checksum    string `json:"checksum"`
}

func (c *Core) registerTools(server *mcp.Server, internalAPI *api.API) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_dataset",
		Description: "Searches the knowledge base for relevant documentation and code examples based on a query string.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchDataset) (*mcp.CallToolResult, any, error) {
		query := args.Query
		queryEmbedding, _ := internalAPI.Llm().GenerateEmbeddings(query)
		results, err := internalAPI.Search(queryEmbedding, 1, args.ConversationID) // Search for 1 result, use ConversationID as context
		if err != nil {
			return nil, nil, err
		}

		if len(results) == 0 {
			return nil, nil, fmt.Errorf("no relevant documents found")
		}

		doc := results[0].Doc // Take the first (most relevant) document

		docOutput := DocumentOutput{
			URL:         doc.URL,
			Title:       doc.Title,
			Description: doc.Description,
			Content:     doc.Content,
			Checksum:    doc.Checksum,
		}

		result, err := json.Marshal(map[string]interface{}{"document": docOutput})
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(result)},
			},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "upsert_document",
		Description: "Adds or updates a document in the knowledge base, automatically generating embeddings.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args UpsertDocumentArgs) (*mcp.CallToolResult, any, error) {
		embeddings, err := internalAPI.Llm().GenerateEmbeddings(args.Content)
		if err != nil {
			return nil, nil, err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(args.Content)))
		doc := &storage.Document{
			URL:         args.URL,
			Title:       args.Title,
			Description: args.Description,
			Content:     args.Content,
			Checksum:    checksum,
			Embeddings:  embeddings,
			Context:     args.ConversationID,
		}
		if err := internalAPI.UpsertDirect(doc); err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Document upserted successfully"}}}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_document",
		Description: "Deletes documents from the knowledge base by URL prefix.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DeleteDocumentArgs) (*mcp.CallToolResult, any, error) {
		err := internalAPI.DeleteDocument(args.URLPrefix)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Documents deleted successfully"}}}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_documents",
		Description: "Lists stored documents in the knowledge base with pagination, optionally filtered by context.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListDocumentsArgs) (*mcp.CallToolResult, any, error) {
		docs, err := internalAPI.ListDocuments(args.ConversationID)
		if err != nil {
			return nil, nil, err
		}
		start := args.Offset
		end := start + args.Limit
		if end > len(docs) {
			end = len(docs)
		}
		paginated := docs[start:end]
		result, err := json.Marshal(map[string]interface{}{"documents": paginated, "total": len(docs)})
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(result)}}}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_document",
		Description: "Retrieves a specific document from the knowledge base by URL.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetDocumentArgs) (*mcp.CallToolResult, any, error) {
		doc, err := internalAPI.GetDocument(args.URL)
		if err != nil {
			return nil, nil, err
		}
		result, err := json.Marshal(doc)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(result)}}}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "learn_api",
		Description: `🚨 MANDATORY FIRST STEP: This tool MUST be called before any other Pons tools.

⚠️ ALL OTHER PONS TOOLS WILL FAIL without a conversationId from this tool.
This tool generates a conversationId that is REQUIRED for all subsequent tool calls. After calling this tool, you MUST extract the conversationId from the response and pass it to every other Pons tool call.

🔄 MULTIPLE CONTEXT SUPPORT: You MUST call this tool multiple times in the same conversation when you need to learn about different documentation contexts. THIS IS NOT OPTIONAL. Just pass the existing conversationId to maintain conversation continuity while loading the new context.

For example, a user might ask a question about the 'admin' context, then switch to the 'functions' context, then ask a question about 'polaris' UI components. In this case, you would call learn_api three times with the following arguments:

- learn_api(api: "admin") -> conversationId: "admin"
- learn_api(api: "functions", conversationId: "admin") -> conversationId: "functions"
- learn_api(api: "polaris", conversationId: "functions") -> conversationId: "polaris"

This is because the conversationId is used to maintain conversation continuity while loading the new context.

🚨 Valid arguments for api are:
    - Any string representing a documentation context (e.g., "shopify-admin", "my-project-docs", "general-knowledge"). This string will be used as the conversationId for subsequent tool calls.

🔄 WORKFLOW:
1. Call learn_api first with the initial API (context)
2. Extract the conversationId from the response
3. Pass that same conversationId to ALL other Pons tools
4. If you need to know more about a different context at any point in the conversation, call learn_api again with the new API (context) and the same conversationId

DON'T SEARCH THE WEB WHEN REFERENCING INFORMATION FROM THIS KNOWLEDGE BASE. IT WILL NOT BE ACCURATE.
PREFER THE USE OF THE search_dataset TOOL TO RETRIEVE INFORMATION FROM THE KNOWLEDGE BASE.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args LearnApiArgs) (*mcp.CallToolResult, any, error) {
		// In Pons, the 'api' directly maps to the 'context' (conversationId)
		// We simply return the 'api' string as the conversationId.
		// If a conversationId is passed, it means we are switching context.

		conversationId := args.Api // The new context is the conversationId

		// You could add logic here to validate if the 'api' (context) exists
		// For now, we'll assume any string is a valid context.

		result, err := json.Marshal(map[string]interface{}{"conversationId": conversationId})
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(result)},
			},
		}, nil, nil
	})
}