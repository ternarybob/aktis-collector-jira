package services

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "aktis-collector-jira/internal/common"
	. "aktis-collector-jira/internal/interfaces"

	"github.com/go-resty/resty/v2"
)

type jiraClient struct {
	client   *resty.Client
	baseURL  string
	username string
	apiToken string
}

func NewJiraClient(config *JiraConfig) JiraClient {
	client := resty.New().
		SetBaseURL(config.BaseURL).
		SetBasicAuth(config.APIConfig.Username, config.APIConfig.APIToken).
		SetTimeout(time.Duration(config.Timeout)*time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	return &jiraClient{
		client:   client,
		baseURL:  config.BaseURL,
		username: config.APIConfig.Username,
		apiToken: config.APIConfig.APIToken,
	}
}

func (jc *jiraClient) SearchIssues(jql string, maxResults, startAt int) (*SearchResponse, error) {
	var response SearchResponse

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

func (jc *jiraClient) BuildJQL(projectKey string, issueTypes, statuses []string, updatedAfter string) string {
	parts := []string{fmt.Sprintf("project = %s", projectKey)}

	if len(issueTypes) > 0 {
		types := make([]string, len(issueTypes))
		for i, t := range issueTypes {
			types[i] = fmt.Sprintf("'%s'", t)
		}
		parts = append(parts, fmt.Sprintf("issuetype in (%s)", strings.Join(types, ", ")))
	}

	if len(statuses) > 0 {
		statusList := make([]string, len(statuses))
		for i, s := range statuses {
			statusList[i] = fmt.Sprintf("'%s'", s)
		}
		parts = append(parts, fmt.Sprintf("status in (%s)", strings.Join(statusList, ", ")))
	}

	if updatedAfter != "" {
		parts = append(parts, fmt.Sprintf("updated >= '%s'", updatedAfter))
	}

	return strings.Join(parts, " AND ")
}
