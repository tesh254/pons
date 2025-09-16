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

type SearchDocChunks struct {
	Query   string `json:"query" jsonschema:"required"`
	Context string `json:"context,omitempty"`
}

type UpsertDocumentArgs struct {
	URL         string `json:"url" jsonschema:"required"`
	Content     string `json:"content" jsonschema:"required"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Context     string `json:"context,omitempty"`
}

type DeleteDocumentArgs struct {
	URLPrefix string `json:"url_prefix" jsonschema:"required"`
	Context   string `json:"context,omitempty"`
}

type ListDocumentsArgs struct {
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	Context string `json:"context,omitempty"`
}

type GetDocumentArgs struct {
	URL     string `json:"url" jsonschema:"required"`
	Context string `json:"context,omitempty"`
}

type LearnApiArgs struct {
	Api     string `json:"api" jsonschema:"required"`
	Context string `json:"context,omitempty"`
}

type GetContextArgs struct {
	Context string `json:"context,omitempty"`
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

// New struct to include score for search results
type SearchOutput struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Content     string  `json:"content"`
	Checksum    string  `json:"checksum"`
	Score       float64 `json:"score"`
}

func (c *Core) registerTools(server *mcp.Server, internalAPI *api.API) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_doc_chunks",
		Description: "Searches the knowledge base for relevant documentation and code examples based on a query string.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchDocChunks) (*mcp.CallToolResult, any, error) {
		query := args.Query
		results, err := internalAPI.Search(query, 3, args.Context) // Pass query string directly
		if err != nil {
			if err.Error() == "no documents found for search" { // Updated error message
				return nil, nil, fmt.Errorf("no relevant documents found")
			}
			return nil, nil, err
		}

		if len(results) == 0 {
			return nil, nil, fmt.Errorf("no relevant documents found")
		}

		var searchOutputs []SearchOutput
		for _, res := range results {
			searchOutputs = append(searchOutputs, SearchOutput{
				URL:         res.Doc.URL,
				Title:       res.Doc.Title,
				Description: res.Doc.Description,
				Content:     res.Doc.Content,
				Checksum:    res.Doc.Checksum,
				Score:       res.Score,
			})
		}

		result, err := json.Marshal(searchOutputs)
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
			Context:     args.Context,
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
		err := internalAPI.DeleteDocument(args.URLPrefix, args.Context)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Documents deleted successfully"}}}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_documents",
		Description: "Lists stored documents in the knowledge base with pagination, optionally filtered by context.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListDocumentsArgs) (*mcp.CallToolResult, any, error) {
		docs, err := internalAPI.ListDocuments(args.Context, args.Limit)
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
		doc, err := internalAPI.GetDocument(args.URL, args.Context)
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
		Name:        "get_contexts",
		Description: "Retrieves a list of unique contexts from the knowledge base.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetContextArgs) (*mcp.CallToolResult, any, error) {
		contexts, err := internalAPI.GetContexts()
		if err != nil {
			return nil, nil, err
		}

		result, err := json.Marshal(map[string]interface{}{"contexts": contexts})
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
