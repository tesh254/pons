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
}

type UpsertDocumentArgs struct {
	URL         string `json:"url" jsonschema:"required"`
	Content     string `json:"content" jsonschema:"required"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type DeleteDocumentArgs struct {
	URLPrefix string `json:"url_prefix" jsonschema:"required"`
}

type ListDocumentsArgs struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type GetDocumentArgs struct {
	URL string `json:"url" jsonschema:"required"`
}

type SearchDatasetTopKArgs struct {
	Query     string  `json:"query" jsonschema:"required"`
	TopK      int     `json:"top_k" jsonschema:"required"`
	Threshold float64 `json:"threshold,omitempty"`
}

func (c *Core) StartServer(internalAPI *api.API, httpAddress string) error {
	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "Pons MCP Server", Version: "v1.0.0"}, nil)

	c.registerTools(server, internalAPI)

	var transport mcp.Transport
	if httpAddress != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("Pons MCP handler listening at %s", httpAddress)
		http.ListenAndServe(httpAddress, loggingHandler(handler))
	} else {
		transport = &mcp.StdioTransport{}
	}

	t := &mcp.LoggingTransport{Transport: transport, Writer: os.Stderr}
	return server.Run(ctx, t)
}

func (c *Core) ServeHTTP(server *mcp.Server, httpAddress string) error {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
	log.Printf("Pons MCP handler listening at %s", httpAddress)
	return http.ListenAndServe(httpAddress, handler)
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
		Description: "Search the dataset for relevant knowledge based on a query string.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchDataset) (*mcp.CallToolResult, any, error) {
		query := args.Query
		queryEmbedding, _ := internalAPI.Llm().GenerateEmbeddings(query)
		doc, _, err := internalAPI.Search(queryEmbedding)
		if err != nil {
			return nil, nil, err
		}

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
		Description: "Upsert a document into the dataset, auto-generating embeddings.",
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
		}
		if err := internalAPI.UpsertDirect(doc); err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Document upserted successfully"}}}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_document",
		Description: "Delete documents by URL prefix.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DeleteDocumentArgs) (*mcp.CallToolResult, any, error) {
		err := internalAPI.DeleteDocument(args.URLPrefix)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Documents deleted successfully"}}}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_documents",
		Description: "List stored documents with pagination.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListDocumentsArgs) (*mcp.CallToolResult, any, error) {
		docs, err := internalAPI.ListDocuments()
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
		Description: "Get a specific document by URL.",
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
}
