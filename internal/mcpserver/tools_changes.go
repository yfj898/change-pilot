package mcpserver

import (
	"context"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/mark3labs/mcp-go/mcp"
)

// BindChanges keeps the upstream server constructor usable for existing callers.
func (s *Server) BindChanges(changes *service.ChangeService) { s.changes = changes }
func changeContextTool() mcp.Tool {
	return mcp.NewTool(types.MCPEndpointToolGetChangeContext, mcp.WithDescription("Read one workspace change's objective, exact Git sources, confirmation and input revision. Does not edit code, confirm a plan or submit execution."), mcp.WithString("change_id", mcp.Required()), readOnlyEndpointAnnotations())
}
func verificationRunTool() mcp.Tool {
	return mcp.NewTool(types.MCPEndpointToolGetVerificationRun, mcp.WithDescription("Read an existing verification run, its actual commit, attempts and mandatory-check verdict. Does not create or retry a run."), mcp.WithString("change_id", mcp.Required()), mcp.WithString("run_id", mcp.Required()), readOnlyEndpointAnnotations())
}
func (s *Server) handleChangeContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.changes == nil {
		return mcp.NewToolResultError("change workspace is unavailable"), nil
	}
	out, err := s.changes.Context(ctx, req.GetString("change_id", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultStructured(out, string(b)), nil
}
func (s *Server) handleVerificationRun(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.changes == nil {
		return mcp.NewToolResultError("change workspace is unavailable"), nil
	}
	out, err := s.changes.Run(ctx, req.GetString("change_id", ""), req.GetString("run_id", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultStructured(out, string(b)), nil
}
