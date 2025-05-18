package github

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v69/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Custom mock pattern for dependency review API endpoint
var GetReposDependencyGraphCompareByOwnerByRepoByBasehead mock.EndpointPattern = mock.EndpointPattern{
	Pattern: "/repos/{owner}/{repo}/dependency-graph/compare/{basehead}",
	Method:  "GET",
}

func Test_GetDependencyReviewCompare(t *testing.T) {
	// Verify tool definition once
	mockClient := github.NewClient(nil)
	tool, _ := GetDependencyReviewCompare(stubGetClientFn(mockClient), translations.NullTranslationHelper)

	assert.Equal(t, "get_dependency_review_compare", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.InputSchema.Properties, "owner")
	assert.Contains(t, tool.InputSchema.Properties, "repo")
	assert.Contains(t, tool.InputSchema.Properties, "basehead")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.ElementsMatch(t, tool.InputSchema.Required, []string{"owner", "repo", "basehead"})

	// Setup mock dependency changes for success case
	mockDependencyChanges := []map[string]interface{}{
		{
			"change_type":           "removed",
			"manifest":              "package.json",
			"ecosystem":             "npm",
			"name":                  "helmet",
			"version":               "4.6.0",
			"package_url":           "pkg:npm/helmet@4.6.0",
			"license":               "MIT",
			"source_repository_url": "https://github.com/helmetjs/helmet",
			"vulnerabilities":       []interface{}{},
		},
		{
			"change_type":           "added",
			"manifest":              "package.json",
			"ecosystem":             "npm",
			"name":                  "helmet",
			"version":               "5.0.0",
			"package_url":           "pkg:npm/helmet@5.0.0",
			"license":               "MIT",
			"source_repository_url": "https://github.com/helmetjs/helmet",
			"vulnerabilities":       []interface{}{},
		},
		{
			"change_type":           "added",
			"manifest":              "Gemfile",
			"ecosystem":             "rubygems",
			"name":                  "ruby-openid",
			"version":               "2.7.0",
			"package_url":           "pkg:gem/ruby-openid@2.7.0",
			"license":               nil,
			"source_repository_url": "https://github.com/openid/ruby-openid",
			"vulnerabilities": []map[string]interface{}{
				{
					"severity":         "critical",
					"advisory_ghsa_id": "GHSA-fqfj-cmh6-hj49",
					"advisory_summary": "Ruby OpenID",
					"advisory_url":     "https://github.com/advisories/GHSA-fqfj-cmh6-hj49",
				},
			},
		},
	}

	tests := []struct {
		name            string
		mockedClient    *http.Client
		requestArgs     map[string]interface{}
		expectError     bool
		expectedChanges []map[string]interface{}
		expectedErrMsg  string
	}{
		{
			name: "successful dependency changes fetch",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					GetReposDependencyGraphCompareByOwnerByRepoByBasehead,
					expectPath(
						t,
						"/repos/owner/repo/dependency-graph/compare/main...feature",
					).andThen(
						mockResponse(t, http.StatusOK, mockDependencyChanges),
					),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":    "owner",
				"repo":     "repo",
				"basehead": "main...feature",
			},
			expectError:     false,
			expectedChanges: mockDependencyChanges,
		},
		{
			name: "successful dependency changes fetch with manifest filter",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					GetReposDependencyGraphCompareByOwnerByRepoByBasehead,
					expectQueryParams(t, map[string]string{
						"name": "package.json",
					}).andThen(
						mockResponse(t, http.StatusOK, mockDependencyChanges[:2]), // Just the npm changes
					),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":    "owner",
				"repo":     "repo",
				"basehead": "main...feature",
				"name":     "package.json",
			},
			expectError:     false,
			expectedChanges: mockDependencyChanges[:2], // Just the npm changes
		},
		{
			name: "dependency changes fetch fails with 404",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					GetReposDependencyGraphCompareByOwnerByRepoByBasehead,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"message": "Not Found"}`))
					}),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":    "owner",
				"repo":     "non-existent-repo",
				"basehead": "main...feature",
			},
			expectError:    true,
			expectedErrMsg: "failed to get dependency changes",
		},
		{
			name: "dependency changes fetch fails with 403",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					GetReposDependencyGraphCompareByOwnerByRepoByBasehead,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte(`{"message": "Dependency review not available for this repository"}`))
					}),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":    "owner",
				"repo":     "repo",
				"basehead": "main...feature",
			},
			expectError:    true,
			expectedErrMsg: "failed to get dependency changes",
		},
		{
			name:         "missing required parameter owner",
			mockedClient: mock.NewMockedHTTPClient(),
			requestArgs: map[string]interface{}{
				"repo":     "repo",
				"basehead": "main...feature",
			},
			expectError:    true,
			expectedErrMsg: "missing required parameter: owner",
		},
		{
			name:         "missing required parameter repo",
			mockedClient: mock.NewMockedHTTPClient(),
			requestArgs: map[string]interface{}{
				"owner":    "owner",
				"basehead": "main...feature",
			},
			expectError:    true,
			expectedErrMsg: "missing required parameter: repo",
		},
		{
			name:         "missing required parameter basehead",
			mockedClient: mock.NewMockedHTTPClient(),
			requestArgs: map[string]interface{}{
				"owner": "owner",
				"repo":  "repo",
			},
			expectError:    true,
			expectedErrMsg: "missing required parameter: basehead",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup client with mock
			client := github.NewClient(tc.mockedClient)
			_, handler := GetDependencyReviewCompare(stubGetClientFn(client), translations.NullTranslationHelper)

			// Create call request
			request := createMCPRequest(tc.requestArgs)

			// Call handler
			result, err := handler(context.Background(), request)

			// Verify results
			if tc.expectError {
				if err != nil {
					assert.Contains(t, err.Error(), tc.expectedErrMsg)
				} else {
					require.NotNil(t, result)
					textContent := getTextResult(t, result)
					assert.Contains(t, textContent.Text, tc.expectedErrMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Parse the result and get the text content
			textContent := getTextResult(t, result)

			// Unmarshal and verify the result
			var returnedChanges []map[string]interface{}
			err = json.Unmarshal([]byte(textContent.Text), &returnedChanges)
			require.NoError(t, err)

			assert.Len(t, returnedChanges, len(tc.expectedChanges))
			for i, change := range returnedChanges {
				assert.Equal(t, tc.expectedChanges[i]["change_type"], change["change_type"])
				assert.Equal(t, tc.expectedChanges[i]["manifest"], change["manifest"])
				assert.Equal(t, tc.expectedChanges[i]["ecosystem"], change["ecosystem"])
				assert.Equal(t, tc.expectedChanges[i]["name"], change["name"])
				assert.Equal(t, tc.expectedChanges[i]["version"], change["version"])
			}
		})
	}
}
