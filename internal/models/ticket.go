package models

// TicketRequest represents the request body for creating a ticket
type TicketRequest struct {
	URL            string                 `json:"url" binding:"required" example:"https://example.com/api/endpoint"`
	Payload        map[string]interface{} `json:"payload" binding:"required"`
	Response       map[string]interface{} `json:"response" binding:"required"`
	RequestHeaders map[string]string      `json:"requestHeaders" binding:"required"`
	ImageS3URL     string                 `json:"imageS3URL" example:"https://bucket.s3.amazonaws.com/screenshot.png"`
}

// TicketResponse represents the response after creating a ticket
type TicketResponse struct {
	TicketID   string `json:"ticketId" example:"PROJECT-123"`
	Status     string `json:"status" example:"created"`
	AssignedTo string `json:"assignedTo" example:"john.doe@company.com"`
	JiraLink   string `json:"jiraLink" example:"https://your-jira.atlassian.net/browse/PROJECT-123"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status" example:"ok"`
	Services  map[string]string `json:"services"`
	Timestamp int64             `json:"timestamp" example:"1647123456"`
}
