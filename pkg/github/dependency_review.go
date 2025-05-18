package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GetDependencyReviewCompare provides a tool to compare dependencies between two commits
func GetDependencyReviewCompare(getClient GetClientFn, t translations.TranslationHelperFunc) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("get_dependency_review_compare",
			mcp.WithDescription(t("TOOL_GET_DEPENDENCY_REVIEW_COMPARE_DESCRIPTION", "Get a diff of the dependencies between commits in a GitHub repository.")),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        t("TOOL_GET_DEPENDENCY_REVIEW_COMPARE_USER_TITLE", "Compare dependencies between commits"),
				ReadOnlyHint: toBoolPtr(true),
			}),
			mcp.WithString("owner",
				mcp.Required(),
				mcp.Description("The account owner of the repository."),
			),
			mcp.WithString("repo",
				mcp.Required(),
				mcp.Description("The name of the repository."),
			),
			mcp.WithString("basehead",
				mcp.Required(),
				mcp.Description("The base and head Git revisions to compare in the format {base}...{head}."),
			),
			mcp.WithString("name",
				mcp.Description("The full path, relative to the repository root, of the dependency manifest file."),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			owner, err := requiredParam[string](request, "owner")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			repo, err := requiredParam[string](request, "repo")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			basehead, err := requiredParam[string](request, "basehead")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			name, err := OptionalParam[string](request, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			client, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitHub client: %w", err)
			}

			// Build the URL for the dependency review API
			url := fmt.Sprintf("repos/%s/%s/dependency-graph/compare/%s", owner, repo, basehead)
			if name != "" {
				// If a manifest file is specified, add it as a query parameter
				url = fmt.Sprintf("%s?name=%s", url, name)
			}

			req, err := client.NewRequest("GET", url, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}

			var dependencyChanges []map[string]interface{}
			resp, err := client.Do(ctx, req, &dependencyChanges)
			if err != nil {
				return nil, fmt.Errorf("failed to get dependency changes: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("failed to read response body: %w", err)
				}
				return mcp.NewToolResultError(fmt.Sprintf("failed to get dependency changes: %s", string(body))), nil
			}

			result, err := json.Marshal(dependencyChanges)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal dependency changes: %w", err)
			}

			return mcp.NewToolResultText(string(result)), nil
		}
}
