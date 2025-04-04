package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/parvez-capri/ronnin/internal/models"
)

type JiraService struct {
	client          *jira.Client
	projectKey      string
	supportTeam     []string
	defaultPriority string
	mongoService    *MongoDBService
}

func NewJiraService(jiraURL, username, apiToken, projectKey string, supportTeam []string, defaultPriority string, mongoService *MongoDBService) (*JiraService, error) {
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: apiToken,
	}

	// Try to create a client and test the connection
	client, err := jira.NewClient(tp.Client(), jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	// Set default priority if empty
	if defaultPriority == "" {
		defaultPriority = "Medium"
	}

	return &JiraService{
		client:          client,
		projectKey:      projectKey,
		supportTeam:     supportTeam,
		defaultPriority: defaultPriority,
		mongoService:    mongoService,
	}, nil
}

func (s *JiraService) CreateTicket(ctx context.Context, req *models.TicketRequest) (*models.TicketResponse, error) {
	// Maximum Jira description length is 32,767 characters
	const maxJiraDescLength = 32000 // Leave some buffer

	// Track if any content is truncated
	wasTruncated := false
	var truncatedContent strings.Builder
	truncatedContent.WriteString("Additional details that couldn't fit in the description:\n\n")

	// Create a better formatted ticket description with clear sections
	description := fmt.Sprintf("h2. Issue Summary\n%s\n\n", req.Payload["issue"])

	// Add a cleaner description section
	if desc, ok := req.Payload["description"].(string); ok && desc != "" {
		description += fmt.Sprintf("h3. Description\n%s\n\n", desc)
	}

	// Add user email and lead ID in a more compact format if available
	var metadataSection string
	if userEmail, ok := req.Payload["userEmail"].(string); ok && userEmail != "" {
		metadataSection += fmt.Sprintf("* *User Email:* %s\n", userEmail)
	}
	if leadID, ok := req.Payload["leadId"].(string); ok && leadID != "" {
		metadataSection += fmt.Sprintf("* *Lead ID:* %s\n", leadID)
	}
	if product, ok := req.Payload["product"].(string); ok && product != "" {
		metadataSection += fmt.Sprintf("* *Product:* %s\n", product)
	}
	if pageURL, ok := req.Payload["url"].(string); ok && pageURL != "" {
		metadataSection += fmt.Sprintf("* *Page URL:* %s\n", pageURL)
	} else if req.URL != "" {
		metadataSection += fmt.Sprintf("* *Page URL:* %s\n", req.URL)
	}

	if metadataSection != "" {
		description += fmt.Sprintf("h3. User Information\n%s\n\n", metadataSection)
	}

	// Add screenshot if available - put it near the top for better visibility
	if req.ImageS3URL != "" && req.ImageS3URL != "None" && req.ImageS3URL != "null" {
		if strings.HasPrefix(req.ImageS3URL, "http") {
			// Add as an image in Jira markdown with expiry note
			description += fmt.Sprintf("h3. Screenshot\n!%s|width=800!\n\n", req.ImageS3URL)
			description += "{panel:title=Note|borderStyle=dashed|borderColor=#ccc|titleBGColor=#f0f0f0|bgColor=#fafafa}\n" +
				"This screenshot URL will expire in 7 days.\n{panel}\n\n"
		} else {
			// Just add as text
			description += fmt.Sprintf("h3. Screenshot\n%s\n\n", req.ImageS3URL)
		}
	}

	// Track remaining characters and length of essential content so far
	essentialLength := len(description)

	// Add creation timestamp
	timestamp := fmt.Sprintf("Ticket created on: %s\n", time.Now().Format(time.RFC1123))
	description += timestamp
	essentialLength += len(timestamp)

	// Calculate remaining characters for dynamic content
	remainingChars := maxJiraDescLength - essentialLength

	// Allocate space for different sections (priorities)
	// Network calls get highest priority - 50% of remaining space
	networkCallsLimit := remainingChars / 2
	// Headers and response get 20% each
	headersLimit := remainingChars / 5
	responseLimit := remainingChars / 5
	// Payload gets the remaining 10%
	payloadLimit := remainingChars - networkCallsLimit - headersLimit - responseLimit

	// Add network calls in a collapsible section if available
	networkCallsSection := ""
	if networkCalls, exists := req.Payload["failedNetworkCalls"]; exists && networkCalls != nil {
		sectionStart := "{panel:title=Failed Network Calls|collapsed=true|borderStyle=solid|borderColor=#ddd|titleBGColor=#f7f7f7|bgColor=#fff}\n"
		sectionEnd := "{panel}\n\n"

		content := ""
		fullContent := ""

		// Try to format as JSON if possible
		if nc, ok := networkCalls.(string); ok {
			// For string representation
			fullContent = "{code:json}\n" + nc + "\n{code}\n"

			if len(nc) > networkCallsLimit-len(sectionStart)-len(sectionEnd)-20 {
				wasTruncated = true
				truncatedContent.WriteString("h3. Complete Network Calls\n")
				truncatedContent.WriteString(fullContent)
				truncatedContent.WriteString("\n\n")

				content += "Network calls data truncated to fit Jira limit:\n"
				content += "{code:json}\n" + nc[:networkCallsLimit-len(sectionStart)-len(sectionEnd)-50] + "\n...[truncated]...\n{code}\n"
			} else {
				content += fullContent
			}
		} else {
			// For structured data, use JSON format
			ncJSON, err := json.Marshal(networkCalls)
			if err == nil {
				fullContent = "{code:json}\n" + string(ncJSON) + "\n{code}\n"

				if len(ncJSON) > networkCallsLimit-len(sectionStart)-len(sectionEnd)-20 {
					wasTruncated = true
					truncatedContent.WriteString("h3. Complete Network Calls\n")
					truncatedContent.WriteString(fullContent)
					truncatedContent.WriteString("\n\n")

					content += "Network calls data truncated to fit Jira limit:\n"
					content += "{code:json}\n" + string(ncJSON[:networkCallsLimit-len(sectionStart)-len(sectionEnd)-50]) + "\n...[truncated]...\n{code}\n"
				} else {
					content += fullContent
				}
			} else {
				content += "Failed to format network calls data as JSON.\n"
			}
		}

		networkCallsSection = sectionStart + content + sectionEnd
	}
	description += networkCallsSection

	// Add technical details in separate collapsible panels
	description += "h3. Technical Details\n\n"

	// Request Headers
	headerSection := ""
	sectionStart := "{panel:title=Request Headers|collapsed=true|borderStyle=solid|borderColor=#ddd|titleBGColor=#f7f7f7|bgColor=#fff}\n"
	sectionEnd := "{panel}\n\n"

	if len(req.RequestHeaders) > 0 {
		content := "{code:json}\n"
		fullContent := "{code:json}\n"

		headersJSON, err := json.MarshalIndent(req.RequestHeaders, "", "  ")
		if err == nil {
			fullContent += string(headersJSON) + "\n{code}\n"

			if len(headersJSON) > headersLimit-len(sectionStart)-len(sectionEnd)-20 {
				wasTruncated = true
				truncatedContent.WriteString("h3. Complete Request Headers\n")
				truncatedContent.WriteString(fullContent)
				truncatedContent.WriteString("\n\n")

				content += string(headersJSON[:headersLimit-len(sectionStart)-len(sectionEnd)-30]) + "\n...[truncated]...\n"
			} else {
				content += string(headersJSON)
			}
		} else {
			// Fallback to simple key-value format with truncation if needed
			headersStr := ""
			fullHeadersStr := ""
			for k, v := range req.RequestHeaders {
				line := fmt.Sprintf("%s: %s\n", k, v)
				fullHeadersStr += line

				if len(headersStr)+len(line) > headersLimit-len(sectionStart)-len(sectionEnd)-30 {
					if headersStr == "" {
						// Ensure at least the first header is included
						headersStr = line
					}
					wasTruncated = true
					truncatedContent.WriteString("h3. Complete Request Headers\n")
					truncatedContent.WriteString("{code}\n" + fullHeadersStr + "{code}\n\n")

					headersStr += "...[truncated]...\n"
					break
				}
				headersStr += line
			}
			content += headersStr
		}
		content += "\n{code}\n"
		headerSection = sectionStart + content + sectionEnd
	} else {
		headerSection = sectionStart + "No request headers available.\n" + sectionEnd
	}
	description += headerSection

	// Response data
	responseSection := ""
	sectionStart = "{panel:title=Response|collapsed=true|borderStyle=solid|borderColor=#ddd|titleBGColor=#f7f7f7|bgColor=#fff}\n"
	sectionEnd = "{panel}\n\n"

	if len(req.Response) > 0 {
		content := "{code:json}\n"
		fullContent := "{code:json}\n"

		responseJSON, err := json.MarshalIndent(req.Response, "", "  ")
		if err == nil {
			fullContent += string(responseJSON) + "\n{code}\n"

			if len(responseJSON) > responseLimit-len(sectionStart)-len(sectionEnd)-20 {
				wasTruncated = true
				truncatedContent.WriteString("h3. Complete Response\n")
				truncatedContent.WriteString(fullContent)
				truncatedContent.WriteString("\n\n")

				content += string(responseJSON[:responseLimit-len(sectionStart)-len(sectionEnd)-30]) + "\n...[truncated]...\n"
			} else {
				content += string(responseJSON)
			}
		} else {
			// Fallback to simple string representation with truncation
			respStr := fmt.Sprintf("%v", req.Response)
			fullContent += respStr + "\n{code}\n"

			if len(respStr) > responseLimit-len(sectionStart)-len(sectionEnd)-30 {
				wasTruncated = true
				truncatedContent.WriteString("h3. Complete Response\n")
				truncatedContent.WriteString(fullContent)
				truncatedContent.WriteString("\n\n")

				content += respStr[:responseLimit-len(sectionStart)-len(sectionEnd)-30] + "\n...[truncated]...\n"
			} else {
				content += respStr
			}
		}
		content += "\n{code}\n"
		responseSection = sectionStart + content + sectionEnd
	} else {
		responseSection = sectionStart + "No response data available.\n" + sectionEnd
	}
	description += responseSection

	// Raw payload data (limited to remaining space)
	payloadSection := ""
	sectionStart = "{panel:title=Full Payload Data|collapsed=true|borderStyle=solid|borderColor=#ddd|titleBGColor=#f7f7f7|bgColor=#fff}\n"
	sectionEnd = "{panel}\n\n"

	content := "{code:json}\n"
	fullContent := "{code:json}\n"

	payloadJSON, err := json.MarshalIndent(req.Payload, "", "  ")
	if err == nil {
		fullContent += string(payloadJSON) + "\n{code}\n"

		if len(payloadJSON) > payloadLimit-len(sectionStart)-len(sectionEnd)-20 {
			wasTruncated = true
			truncatedContent.WriteString("h3. Complete Payload\n")
			truncatedContent.WriteString(fullContent)
			truncatedContent.WriteString("\n\n")

			content += string(payloadJSON[:payloadLimit-len(sectionStart)-len(sectionEnd)-30]) + "\n...[truncated]...\n"
		} else {
			content += string(payloadJSON)
		}
	} else {
		// Fallback to simple string representation with truncation
		payloadStr := fmt.Sprintf("%v", req.Payload)
		fullContent += payloadStr + "\n{code}\n"

		if len(payloadStr) > payloadLimit-len(sectionStart)-len(sectionEnd)-30 {
			wasTruncated = true
			truncatedContent.WriteString("h3. Complete Payload\n")
			truncatedContent.WriteString(fullContent)
			truncatedContent.WriteString("\n\n")

			content += payloadStr[:payloadLimit-len(sectionStart)-len(sectionEnd)-30] + "\n...[truncated]...\n"
		} else {
			content += payloadStr
		}
	}
	content += "\n{code}\n"
	payloadSection = sectionStart + content + sectionEnd
	description += payloadSection

	// Final check to ensure we're under limit
	if len(description) > maxJiraDescLength {
		// If still too long, truncate the whole thing
		wasTruncated = true
		truncatedContent.WriteString("h3. Full Original Description\n")
		truncatedContent.WriteString(description)
		truncatedContent.WriteString("\n\n")

		description = description[:maxJiraDescLength-100] + "\n\n[Content truncated due to Jira character limit. See comments for complete information.]"
	}

	// Get random team member for assignment
	assignee := s.getRandomTeamMember()

	// Get available issue types for the project to find the Bug type
	issueTypeID := ""
	metaProject, _, err := s.client.Issue.GetCreateMeta(s.projectKey)
	if err != nil {
		// Use default issue type ID if we can't get metadata
		issueTypeID = "10001" // Common default for Bug in Jira Cloud
	} else if metaProject != nil && len(metaProject.Projects) > 0 {
		for _, project := range metaProject.Projects {
			if project.Key == s.projectKey {
				for _, issueType := range project.IssueTypes {
					if issueType.Name == "Bug" {
						issueTypeID = issueType.Id
					}
				}
			}
		}
	}

	// If we couldn't find the Bug type, use a default
	if issueTypeID == "" {
		issueTypeID = "10001" // Common default for Bug in Jira Cloud
	}

	// Create Jira issue
	issueFields := &jira.IssueFields{
		Project: jira.Project{
			Key: s.projectKey,
		},
		Summary:     fmt.Sprintf("Issue Report: %s", req.Payload["issue"]),
		Description: description,
		Type: jira.IssueType{
			ID: issueTypeID,
		},
		Assignee: &jira.User{
			AccountID: assignee,
		},
		// Removing Priority field as it's not available on the issue creation screen
		// Priority: &jira.Priority{
		//     Name: s.defaultPriority,
		// },
	}

	issue := &jira.Issue{
		Fields: issueFields,
	}

	// Log the data being sent to Jira API
	fmt.Printf("\n=== JIRA TICKET DATA ===\n")
	fmt.Printf("Project Key: %s\n", s.projectKey)
	fmt.Printf("Issue Type ID: %s\n", issueTypeID)
	fmt.Printf("Summary: %s\n", issueFields.Summary)
	fmt.Printf("Assignee Account ID: %s\n", assignee)

	// Log S3 image URL if present
	if req.ImageS3URL != "" && req.ImageS3URL != "None" && req.ImageS3URL != "null" {
		fmt.Printf("Image URL (presigned, expires in 7 days): %s\n", req.ImageS3URL)
	} else {
		fmt.Printf("Image URL: No valid image URL provided\n")
	}

	// Log first 500 chars of description (as it might be very long)
	descLen := len(description)
	if descLen > 500 {
		fmt.Printf("Description (first 500 chars): %s...\n", description[:500])
	} else {
		fmt.Printf("Description: %s\n", description)
	}

	// Log payload details
	fmt.Printf("=== Payload Details ===\n")
	for k, v := range req.Payload {
		// For values that might be long (like description), truncate them
		strVal := fmt.Sprintf("%v", v)
		if len(strVal) > 100 {
			strVal = strVal[:100] + "..."
		}
		fmt.Printf("  %s: %s\n", k, strVal)
	}

	// Log headers
	fmt.Printf("=== Request Headers ===\n")
	for k, v := range req.RequestHeaders {
		fmt.Printf("  %s: %s\n", k, v)
	}

	fmt.Printf("=== END JIRA TICKET DATA ===\n\n")

	// Update to use context in the Create call if the client supports it
	newIssue, resp, err := s.client.Issue.Create(issue)
	if err != nil {
		// Log detailed error information
		statusCode := 0
		var responseBody string

		if resp != nil {
			statusCode = resp.StatusCode
			if resp.Body != nil {
				// Try to read response body for better error details
				bodyBytes := make([]byte, 1024)
				n, _ := resp.Body.Read(bodyBytes)
				if n > 0 {
					responseBody = string(bodyBytes[:n])
				}
				resp.Body.Close()
			}
		}

		// Return error with more details
		return nil, fmt.Errorf("failed to create Jira ticket: status=%d, error=%v, response=%s",
			statusCode, err, responseBody)
	}

	// Fix the URL string conversion
	baseURL := &url.URL{
		Scheme: "https",
		Host:   s.client.GetBaseURL().Host,
	}

	ticketResponse := &models.TicketResponse{
		TicketID:   newIssue.Key,
		Status:     "created",
		AssignedTo: assignee,
		JiraLink:   fmt.Sprintf("%s/browse/%s", baseURL.String(), newIssue.Key),
	}

	// If content was truncated, add it as a comment
	if wasTruncated {
		commentBody := truncatedContent.String()

		// Check if the comment itself is too long (32,767 characters max)
		if len(commentBody) > maxJiraDescLength {
			commentBody = commentBody[:maxJiraDescLength-100] + "\n\n[Comment truncated due to Jira character limit]"
		}

		comment := &jira.Comment{
			Body: commentBody,
		}

		fmt.Printf("Adding a comment with truncated content to ticket %s\n", newIssue.Key)

		_, _, err := s.client.Issue.AddComment(newIssue.Key, comment)
		if err != nil {
			// Log error but don't fail the ticket creation
			fmt.Printf("Failed to add comment with truncated content: %v\n", err)
		} else {
			fmt.Printf("Successfully added comment with truncated content\n")
		}
	}

	// Save the ticket to MongoDB if available
	if s.mongoService != nil {
		// Create flattened ticket object
		flattenedTicket := &FlattenedTicket{
			TicketID:   newIssue.Key,
			Status:     "created",
			AssignedTo: assignee,
			JiraLink:   fmt.Sprintf("%s/browse/%s", baseURL.String(), newIssue.Key),
			CreatedAt:  time.Now(),
		}

		// Extract basic fields
		if issueValue, ok := req.Payload["issue"].(string); ok {
			flattenedTicket.Issue = issueValue
		}
		if descValue, ok := req.Payload["description"].(string); ok {
			flattenedTicket.Description = descValue
		}
		if emailValue, ok := req.Payload["userEmail"].(string); ok {
			flattenedTicket.UserEmail = emailValue
		}
		if leadValue, ok := req.Payload["leadId"].(string); ok {
			flattenedTicket.LeadID = leadValue
		}
		if productValue, ok := req.Payload["product"].(string); ok {
			flattenedTicket.Product = productValue
		}

		// Set page URL
		if pageURL, ok := req.Payload["url"].(string); ok {
			flattenedTicket.PageURL = pageURL
		} else {
			flattenedTicket.PageURL = req.URL
		}

		// Set image URL
		if req.ImageS3URL != "" && req.ImageS3URL != "None" && req.ImageS3URL != "null" {
			flattenedTicket.ImageURL = req.ImageS3URL
		}

		// Serialize complex data to JSON strings
		if networkCalls, exists := req.Payload["failedNetworkCalls"]; exists {
			networkCallsJSON, err := json.Marshal(networkCalls)
			if err == nil {
				flattenedTicket.FailedNetworkCallsJSON = string(networkCallsJSON)
			} else {
				// Try as string
				if ncStr, ok := networkCalls.(string); ok {
					flattenedTicket.FailedNetworkCallsJSON = ncStr
				}
			}
		}

		// Convert payload to JSON string
		payloadJSON, err := json.Marshal(req.Payload)
		if err == nil {
			flattenedTicket.PayloadJSON = string(payloadJSON)
		}

		// Convert response to JSON string
		responseJSON, err := json.Marshal(req.Response)
		if err == nil {
			flattenedTicket.ResponseJSON = string(responseJSON)
		}

		// Convert headers to JSON string
		headersJSON, err := json.Marshal(req.RequestHeaders)
		if err == nil {
			flattenedTicket.RequestHeadersJSON = string(headersJSON)
		}

		// Save to MongoDB
		fmt.Printf("Saving ticket %s to MongoDB\n", newIssue.Key)
		mongoID, err := s.mongoService.SaveTicket(ctx, flattenedTicket)
		if err != nil {
			// Log error but don't fail the ticket creation
			fmt.Printf("Failed to save ticket to MongoDB: %v\n", err)
		} else {
			fmt.Printf("Successfully saved ticket to MongoDB with ID: %s\n", mongoID)
		}
	}

	return ticketResponse, nil
}

func (s *JiraService) getRandomTeamMember() string {
	// If there are no team members, return empty string
	if len(s.supportTeam) == 0 {
		return ""
	}

	// Get random index using math/rand
	// Note: In Go 1.20+, we don't need to call rand.Seed
	randIndex := rand.Intn(len(s.supportTeam))
	selectedMember := s.supportTeam[randIndex]

	fmt.Printf("Randomly selected team member %d of %d: %s\n",
		randIndex+1, len(s.supportTeam), selectedMember)

	return selectedMember
}

// Add a method for cleanup if needed
func (s *JiraService) Cleanup() error {
	// Add any cleanup logic here
	return nil
}

// Helper function to get keys from a map for logging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getStringMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetMongoService returns the MongoDB service
func (s *JiraService) GetMongoService() *MongoDBService {
	return s.mongoService
}
