package core

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tesh254/pons/internal/api"
	"github.com/tesh254/pons/internal/llm"
)

type Core struct {
}

type Content struct {
	Title       string  `json:"title" jsonschema:"required,description=The title to submit"`
	Description *string `json:"description" jsonschema:"description=The description to submit"`
}

type SearchDataset struct {
	Query string `json:"query" jsonschema:"required"`
}

func (c *Core) StartServer(internalAPI *api.API) error {
	ctx := context.Background()
	_, serverTransport := mcp.NewInMemoryTransports()

	server := mcp.NewServer(&mcp.Implementation{Name: "Pons MCP Server", Version: "v1.0.0"}, nil)

	c.registerSearchDataset(server, internalAPI)

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	defer serverSession.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (c *Core) registerSearchDataset(server *mcp.Server, internalAPI *api.API) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_dataset",
		Description: `search_dataset when provided a string query will look up the current dataset store locally for a tool`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchDataset) (*mcp.CallToolResult, any, error) {
		query := args.Query
		var embeddings llm.Embeddings
		queryEmbedding, _ := embeddings.GenerateEmbeddings(query)
		doc, _, err := internalAPI.Search(queryEmbedding)
		if err != nil {
			return nil, nil, err
		}

		result, err := json.Marshal(map[string]interface{}{"document": doc})
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
