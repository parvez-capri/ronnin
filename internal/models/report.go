package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ReportIssueRequest represents the form data for reporting an issue
type ReportIssueRequest struct {
	Issue              string `form:"issue" binding:"required"`
	Description        string `form:"description" binding:"required"`
	UserEmail          string `form:"userEmail"`
	LeadID             string `form:"leadId"`
	Product            string `form:"product"`
	FailedNetworkCalls string `form:"failedNetworkCalls"`
	PageURL            string `form:"pageUrl"`
	ImageS3URL         string `form:"imageS3URL"`
}

// GetNetworkCalls parses the FailedNetworkCalls string into []NetworkCall
func (r *ReportIssueRequest) GetNetworkCalls() ([]NetworkCall, error) {
	var calls []NetworkCall

	if r.FailedNetworkCalls == "" {
		return calls, nil
	}

	// The input might be a JSON string that needs to be processed
	// First, try direct unmarshal to see if it's already valid JSON
	if err := json.Unmarshal([]byte(r.FailedNetworkCalls), &calls); err == nil {
		return calls, nil
	}

	// If direct unmarshal failed, the string might be escaped JSON
	// First, remove any surrounding quotes if present (common issue with form data)
	input := strings.TrimSpace(r.FailedNetworkCalls)
	if (strings.HasPrefix(input, "\"") && strings.HasSuffix(input, "\"")) ||
		(strings.HasPrefix(input, "'") && strings.HasSuffix(input, "'")) {
		// Remove the outer quotes
		input = input[1 : len(input)-1]
	}

	// Replace escaped quotes with actual quotes
	input = strings.ReplaceAll(input, "\\\"", "\"")

	// Replace escaped backslashes with actual backslashes
	input = strings.ReplaceAll(input, "\\\\", "\\")

	// Try to parse the cleaned string
	if err := json.Unmarshal([]byte(input), &calls); err == nil {
		return calls, nil
	}

	// If the input looks like a stringified JSON array
	// Try one more approach - sometimes JSON is double stringified
	var decodedString string
	if err := json.Unmarshal([]byte(input), &decodedString); err == nil {
		// If we successfully extracted a string, try to parse that as JSON
		if err := json.Unmarshal([]byte(decodedString), &calls); err == nil {
			return calls, nil
		}
	}

	// As a fallback for debugging, log the input pattern
	fmt.Printf("Failed to parse network calls. Input sample (first 100 chars): %s\n",
		input[:min(len(input), 100)])

	// If all parsing attempts fail, return an empty array instead of failing
	// We'll handle the raw string separately in the handler
	return calls, fmt.Errorf("could not parse network calls after multiple attempts")
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NetworkCall represents a failed network request
type NetworkCall struct {
	RequestData struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    interface{}       `json:"body"`
	} `json:"requestData"`
	ResponseStatus  int    `json:"responseStatus"`
	ResponseHeaders string `json:"responseHeaders"`
	ResponseBody    string `json:"responseBody"`
	PageURL         string `json:"pageUrl"`
	Timestamp       string `json:"timestamp"`
}

