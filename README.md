# pons
Turn any public documentation link into an MCP server for your local AI tool. Fast, fun, and effortless—just point and connect.

## Using the Pons MCP Server

The Pons MCP server allows your local AI tools to connect and utilize its capabilities. To start the server, use the `pons start` command:

```bash
pons start
```

By default, the server listens on `http://localhost:8080`. You can specify a different address and port using the `--http-address` flag:

```bash
pons start --http-address "0.0.0.0:8081"
```

### Connecting Your AI Tool

To connect your AI tool to the Pons MCP server, configure your tool to use the server's address. For example, if your AI tool supports connecting to an MCP server, you would typically provide the `http://localhost:8080` (or your custom address) as the server endpoint.

Refer to your AI tool's documentation for specific instructions on how to configure an MCP server connection.
