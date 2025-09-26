package collector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// JiraClient handles Jira API interactionstype JiraClient struct {
	client   *resty.Client
	baseURL  string
	username string
	apiToken string
}

// JiraIssue represents a Jira ticket/issuetype JiraIssue struct {
	Key     string                 `json:"key"`
	ID      string                 `json:"id"`
	Self    string                 `json:"self"`
	Fields  map[string]interface{} `json:"fields"`
}

// JiraSearchResponse represents the Jira search API responsetype JiraSearchResponse struct {
	StartAt    int          `json:"startAt"`
	MaxResults int          `json:"maxResults"`
	Total      int          `json:"total"`
	Issues     []JiraIssue  `json:"issues"`
}

// NewJiraClient creates a new Jira API clientfunc NewJiraClient(config *JiraConfig) *JiraClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetBasicAuth(config.Username, config.APIToken).
		SetTimeout(time.Duration(config.Timeout) * time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	return &JiraClient{
		client:   client,
		baseURL:  config.BaseURL,
		username: config.Username,
		apiToken: config.APIToken,
	}
}

// SearchIssues searches for Jira issues using JQLfunc (jc *JiraClient) SearchIssues(jql string, startAt int, maxResults int) (*JiraSearchResponse, error) {
	var response JiraSearchResponse
	
	resp, err := jc.client.R().
		SetQueryParam("jql", jql).
		SetQueryParam("startAt", strconv.Itoa(startAt)).
		SetQueryParam("maxResults", strconv.Itoa(maxResults)).
		SetQueryParam("fields", "*all").
		SetResult(&response).
		Get("/rest/api/3/search")

	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Jira API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &response, nil
}

// GetIssue retrieves a single issue by keyfunc (jc *JiraClient) GetIssue(key string) (*JiraIssue, error) {
	var issue JiraIssue
	
	resp, err := jc.client.R().
		SetQueryParam("fields", "*all").
		SetResult(&issue).
		Get(fmt.Sprintf("/rest/api/3/issue/%s", key))

	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", key, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Jira API returned status %d for issue %s", resp.StatusCode(), key)
	}

	return &issue, nil
}

// BuildJQL constructs a JQL query for a project and filtersfunc BuildJQL(project ProjectConfig, updatedSince time.Time) string {
	parts := []string{fmt.Sprintf("project = %s", project.Key)}

	// Add issue type filter
	if len(project.IssueTypes) > 0 {
		types := make([]string, len(project.IssueTypes))
		for i, t := range project.IssueTypes {
			types[i] = fmt.Sprintf("'%s'", t)
		}
		parts = append(parts, fmt.Sprintf("issuetype in (%s)", strings.Join(types, ", ")))
	}

	// Add status filter
	if len(project.Statuses) > 0 {
		statuses := make([]string, len(project.Statuses))
		for i, s := range project.Statuses {
			statuses[i] = fmt.Sprintf("'%s'", s)
		}
		parts = append(parts, fmt.Sprintf("status in (%s)", strings.Join(statuses, ", ")))
	}

	// Add update time filter for incremental updates
	if !updatedSince.IsZero() {
		parts = append(parts, fmt.Sprintf("updated >= '%s'", updatedSince.Format("2006-01-02 15:04")))
	}

	return strings.Join(parts, " AND ")
}

// ExtractIssueData extracts relevant data from a Jira issuefunc ExtractIssueData(issue *JiraIssue) map[string]interface{} {
	data := make(map[string]interface{})
	
	// Basic fields
	data["key"] = issue.Key
	data["id"] = issue.ID
	data["self"] = issue.Self
	
	// Extract common fields
	if summary, ok := issue.Fields["summary"].(string); ok {
		data["summary"] = summary
	}
	
	if description, ok := issue.Fields["description"].(string); ok {
		data["description"] = description
	}
	
	if issueType, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
		if name, ok := issueType["name"].(string); ok {
			data["issue_type"] = name
		}
	}
	
	if status, ok := issue.Fields["status"].(map[string]interface{}); ok {
		if name, ok := status["name"].(string); ok {
			data["status"] = name
		}
	}
	
	if priority, ok := issue.Fields["priority"].(map[string]interface{}); ok {
		if name, ok := priority["name"].(string); ok {
			data["priority"] = name
		}
	}
	
	if assignee, ok := issue.Fields["assignee"].(map[string]interface{}); ok {
		if name, ok := assignee["displayName"].(string); ok {
			data["assignee"] = name
		} else if email, ok := assignee["emailAddress"].(string); ok {
			data["assignee"] = email
		}
	}
	
	if reporter, ok := issue.Fields["reporter"].(map[string]interface{}); ok {
		if name, ok := reporter["displayName"].(string); ok {
			data["reporter"] = name
		} else if email, ok := reporter["emailAddress"].(string); ok {
			data["reporter"] = email
		}
	}
	
	if created, ok := issue.Fields["created"].(string); ok {
		data["created"] = created
	}
	
	if updated, ok := issue.Fields["updated"].(string); ok {
		data["updated"] = updated
	}
	
	if project, ok := issue.Fields["project"].(map[string]interface{}); ok {
		if key, ok := project["key"].(string); ok {
			data["project_key"] = key
		}
		if name, ok := project["name"].(string); ok {
			data["project_name"] = name
		}
	}
	
	// Extract custom fields
	if customFields, ok := issue.Fields["customfield_10001"]; ok {
		data["custom_field_10001"] = customFields
	}
	
	// Extract labels
	if labels, ok := issue.Fields["labels"].([]interface{}); ok {
		data["labels"] = labels
	}
	
	// Extract components
	if components, ok := issue.Fields["components"].([]interface{}); ok {
		data["components"] = components
	}
	
	// Extract fix versions
	if fixVersions, ok := issue.Fields["fixVersions"].([]interface{}); ok {
		data["fix_versions"] = fixVersions
	}
	
	// Extract affected versions
	if versions, ok := issue.Fields["versions"].([]interface{}); ok {
		data["versions"] = versions
	}
	
	// Store raw fields for reference
	data["raw_fields"] = issue.Fields
	
	return data
}

// GetIssueChangelog retrieves the changelog for an issuefunc (jc *JiraClient) GetIssueChangelog(key string) ([]interface{}, error) {
	var changelog map[string]interface{}
	
	resp, err := jc.client.R().
		SetResult(&changelog).
		Get(fmt.Sprintf("/rest/api/3/issue/%s/changelog", key))

	if err != nil {
		return nil, fmt.Errorf("failed to get changelog for issue %s: %w", key, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Jira API returned status %d for changelog of issue %s", resp.StatusCode(), key)
	}

	if values, ok := changelog["values"].([]interface{}); ok {
		return values, nil
	}

	return []interface{}{}, nil
}