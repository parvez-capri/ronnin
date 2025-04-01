package services

import (
	"context"
	"fmt"
	"net/url"

	jira "github.com/andygrunwald/go-jira"
	"github.com/parvez-capri/ronnin/internal/models"
)

type JiraService struct {
	client      *jira.Client
	projectKey  string
	supportTeam []string
}

func NewJiraService(jiraURL, username, apiToken, projectKey string, supportTeam []string) (*JiraService, error) {
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: apiToken,
	}

	client, err := jira.NewClient(tp.Client(), jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return &JiraService{
		client:      client,
		projectKey:  projectKey,
		supportTeam: supportTeam,
	}, nil
}

func (s *JiraService) CreateTicket(ctx context.Context, req *models.TicketRequest) (*models.TicketResponse, error) {
	// Create ticket description - fixing nested backticks issue
	description := fmt.Sprintf("URL: %s\n\n"+
		"Payload:\n"+
		"```json\n"+
		"%v\n"+
		"```\n\n"+
		"Response:\n"+
		"```json\n"+
		"%v\n"+
		"```\n\n"+
		"Request Headers:\n"+
		"```json\n"+
		"%v\n"+
		"```\n\n"+
		"Screenshot: %s",
		req.URL, req.Payload, req.Response, req.RequestHeaders, req.ImageS3URL)

	// Get random team member for assignment
	assignee := s.getRandomTeamMember()

	// Create Jira issue
	issueFields := &jira.IssueFields{
		Project: jira.Project{
			Key: s.projectKey,
		},
		Summary:     fmt.Sprintf("Support Request: %s", req.URL),
		Description: description,
		Type: jira.IssueType{
			Name: "Bug",
		},
		Assignee: &jira.User{
			Name: assignee,
		},
	}

	issue := &jira.Issue{
		Fields: issueFields,
	}

	// Update to use context in the Create call if the client supports it
	newIssue, _, err := s.client.Issue.Create(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira ticket: %w", err)
	}

	// Fix the URL string conversion
	baseURL := &url.URL{
		Scheme: "https",
		Host:   s.client.GetBaseURL().Host,
	}

	return &models.TicketResponse{
		TicketID:   newIssue.Key,
		Status:     "created",
		AssignedTo: assignee,
		JiraLink:   fmt.Sprintf("%s/browse/%s", baseURL.String(), newIssue.Key),
	}, nil
}

func (s *JiraService) getRandomTeamMember() string {
	// Simple round-robin or random selection
	// In production, you might want to implement a more sophisticated selection algorithm
	if len(s.supportTeam) == 0 {
		return ""
	}
	// For demo, just return the first team member
	return s.supportTeam[0]
}

// Add a method for cleanup if needed
func (s *JiraService) Cleanup() error {
	// Add any cleanup logic here
	return nil
}
