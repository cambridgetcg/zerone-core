package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/toolbox/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// queryServer implements types.QueryServer.
type queryServer struct {
	types.UnimplementedQueryServer
	k Keeper
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{k: keeper}
}

var _ types.QueryServer = &queryServer{}

// Tool returns a single tool by ID.
func (qs *queryServer) Tool(ctx context.Context, req *types.QueryToolRequest) (*types.QueryToolResponse, error) {
	if req == nil || req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}

	tool, ok := qs.k.GetTool(ctx, req.ToolId)
	if !ok {
		return nil, status.Error(codes.NotFound, "tool not found")
	}

	return &types.QueryToolResponse{Tool: tool}, nil
}

// ToolsByDeployer returns all tools for a deployer.
func (qs *queryServer) ToolsByDeployer(ctx context.Context, req *types.QueryByDeployerRequest) (*types.QueryByDeployerResponse, error) {
	if req == nil || req.Deployer == "" {
		return nil, status.Error(codes.InvalidArgument, "deployer is required")
	}

	tools := qs.k.GetToolsByDeployer(ctx, req.Deployer)
	return &types.QueryByDeployerResponse{Tools: tools}, nil
}

// ToolsByCategory returns all tools in a category.
func (qs *queryServer) ToolsByCategory(ctx context.Context, req *types.QueryByCategoryRequest) (*types.QueryByCategoryResponse, error) {
	if req == nil || req.Category == "" {
		return nil, status.Error(codes.InvalidArgument, "category is required")
	}

	tools := qs.k.GetToolsByCategory(ctx, req.Category)
	return &types.QueryByCategoryResponse{Tools: tools}, nil
}

// TrustScore returns the trust snapshot for a tool.
func (qs *queryServer) TrustScore(ctx context.Context, req *types.QueryTrustScoreRequest) (*types.QueryTrustScoreResponse, error) {
	if req == nil || req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}

	snap, ok := qs.k.GetTrustSnapshot(ctx, req.ToolId)
	if !ok {
		return nil, status.Error(codes.NotFound, "trust snapshot not found")
	}

	return &types.QueryTrustScoreResponse{Snapshot: snap}, nil
}

// DependencyTree returns the dependency tree for a tool.
func (qs *queryServer) DependencyTree(ctx context.Context, req *types.QueryDependencyTreeRequest) (*types.QueryDependencyTreeResponse, error) {
	if req == nil || req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}

	tree := qs.k.buildDependencyTree(ctx, req.ToolId, make(map[string]bool), 0, 20)
	return &types.QueryDependencyTreeResponse{Tree: tree}, nil
}

// buildDependencyTree recursively builds the dependency tree.
func (k Keeper) buildDependencyTree(ctx context.Context, toolID string, visited map[string]bool, depth, maxDepth int) *types.DependencyTreeNode {
	if depth > maxDepth || visited[toolID] {
		return &types.DependencyTreeNode{ToolId: toolID, Name: "(circular/depth limit)"}
	}
	visited[toolID] = true

	tool, ok := k.GetTool(ctx, toolID)
	if !ok {
		return &types.DependencyTreeNode{ToolId: toolID, Name: "(not found)"}
	}

	node := &types.DependencyTreeNode{
		ToolId:       tool.Id,
		Name:         tool.Name,
		Version:      tool.Version,
		PricePerCall: tool.PricePerCall,
		TrustScore:   tool.TrustScore,
		Status:       tool.Status,
	}

	for _, depID := range tool.DependencyIds {
		child := k.buildDependencyTree(ctx, depID, visited, depth+1, maxDepth)
		node.Children = append(node.Children, child)
	}

	delete(visited, toolID) // Allow tool to appear in separate branches.
	return node
}

// FreeAllowance returns the free tier allowance for a caller.
func (qs *queryServer) FreeAllowance(ctx context.Context, req *types.QueryFreeAllowanceRequest) (*types.QueryFreeAllowanceResponse, error) {
	if req == nil || req.Caller == "" {
		return nil, status.Error(codes.InvalidArgument, "caller is required")
	}

	fa := qs.k.GetFreeAllowance(ctx, req.Caller)
	return &types.QueryFreeAllowanceResponse{Allowance: fa}, nil
}

// Params returns the current module parameters.
func (qs *queryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params := qs.k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
