package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/parvez-capri/ronnin/internal/models"
	"github.com/parvez-capri/ronnin/internal/services"
	"go.uber.org/zap"
)

type ReportHandler struct {
	jiraService *services.JiraService
	s3Service   *services.S3Service
	logger      *zap.Logger
	validate    *validator.Validate
}

func NewReportHandler(js *services.JiraService, s3s *services.S3Service, log *zap.Logger, validate *validator.Validate) *ReportHandler {
	return &ReportHandler{
		jiraService: js,
		s3Service:   s3s,
		logger:      log,
		validate:    validate,
	}
}

// ReportIssue godoc
// @Summary      Report an issue
// @Description  Creates a JIRA ticket for a reported issue with screenshots and network calls
// @Tags         reports
// @Accept       multipart/form-data
// @Produce      json
// @Param        issue formData string true "Issue title"
// @Param        description formData string true "Issue description"
// @Param        userEmail formData string false "User email"
// @Param        leadId formData string false "Lead ID"
// @Param        product formData string false "Product name"
// @Param        failedNetworkCalls formData string false "Failed network calls JSON"
// @Param        image0 formData file false "Screenshot"
// @Success      201  {object}  models.TicketResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /report-issue [post]
func (h *ReportHandler) ReportIssue(c *gin.Context) {
	var req models.ReportIssueRequest

	// Parse form data with detailed error logging
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Error("Failed to bind request",
			zap.Error(err),
			zap.String("issue", c.PostForm("issue")),
			zap.String("description", c.PostForm("description")),
			zap.String("userEmail", c.PostForm("userEmail")),
			zap.String("leadId", c.PostForm("leadId")),
			zap.String("product", c.PostForm("product")),
			zap.String("failedNetworkCalls", c.PostForm("failedNetworkCalls")),
		)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Validation failed",
			Details: err.Error(),
		})
		return
	}

	// Handle file upload
	file, err := c.FormFile("image0")
	var imageURL string = "" // Initialize with empty string

	// Log raw form data for debugging
	fmt.Printf("\n=== RAW FORM DATA ===\n")
	fmt.Printf("Raw image0 form value: %v\n", c.Request.FormValue("image0"))
	fmt.Printf("Has file? %v\n", file != nil)
	if err != nil {
		fmt.Printf("FormFile error: %s\n", err)
	}
	formFile := c.Request.MultipartForm
	if formFile != nil && formFile.File != nil {
		fmt.Printf("Multipart form files: %v\n", formFile.File)
		if files, exists := formFile.File["image0"]; exists {
			fmt.Printf("Number of image0 files: %d\n", len(files))
			for i, f := range files {
				fmt.Printf("File %d: %s, size: %d\n", i, f.Filename, f.Size)
			}
		} else {
			fmt.Printf("No 'image0' found in multipart form\n")
		}
	} else {
		fmt.Printf("No multipart form or empty form.File\n")
	}
	fmt.Printf("=== END RAW FORM DATA ===\n\n")

	if err == nil && file != nil {
		if h.s3Service != nil {
			// Upload to S3
			imageURL, err = h.s3Service.UploadFile(c.Request.Context(), file)
			if err != nil {
				h.logger.Error("Failed to upload file to S3", zap.Error(err))
				// Continue with the request, just without the image
				imageURL = "" // Set to empty string if upload fails
			} else {
				h.logger.Info("File uploaded to S3 successfully", zap.String("url", imageURL))
			}
		} else {
			// S3 service not available
			h.logger.Warn("S3 service not available, using placeholder URL")
			imageURL = "https://example.com/placeholder.png"
		}
	} else {
		h.logger.Info("No file uploaded or error getting file", zap.Error(err))
	}

	// Parse network calls
	networkCalls, err := req.GetNetworkCalls()
	if err != nil {
		// Log the error but continue with the request
		h.logger.Warn("Processing network calls with fallback approach",
			zap.Error(err),
			zap.String("failedNetworkCalls", req.FailedNetworkCalls[:min(len(req.FailedNetworkCalls), 100)]),
		)

		// Try a more direct approach for debugging - marshal the string directly to the payload
		var rawNetworkData interface{}
		if jsonErr := json.Unmarshal([]byte(req.FailedNetworkCalls), &rawNetworkData); jsonErr == nil {
			// Successfully parsed as generic JSON
			h.logger.Info("Successfully parsed network calls as generic JSON")

			// Create ticket request with parsed JSON
			ticketReq := &models.TicketRequest{
				URL: req.PageURL,
				Payload: map[string]interface{}{
					"issue":              req.Issue,
					"description":        req.Description,
					"userEmail":          req.UserEmail,
					"leadId":             req.LeadID,
					"product":            req.Product,
					"failedNetworkCalls": rawNetworkData,
				},
				Response: map[string]interface{}{
					"status": "reported",
				},
				RequestHeaders: map[string]string{
					"Content-Type": "multipart/form-data",
				},
				ImageS3URL: imageURL,
			}

			// Create ticket with the parsed generic JSON
			response, err := h.jiraService.CreateTicket(c.Request.Context(), ticketReq)
			if err != nil {
				h.logger.Error("Failed to create ticket", zap.Error(err))
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "Failed to create ticket",
					Details: err.Error(),
				})
				return
			}

			c.JSON(http.StatusCreated, response)
			return
		}

		// Use empty array for structured network calls
		networkCalls = []models.NetworkCall{}
	}

	// Create ticket request with original approach (either successful parse or empty array)
	ticketReq := &models.TicketRequest{
		URL: req.PageURL,
		Payload: map[string]interface{}{
			"issue":               req.Issue,
			"description":         req.Description,
			"userEmail":           req.UserEmail,
			"leadId":              req.LeadID,
			"product":             req.Product,
			"failedNetworkCalls":  networkCalls,
			"rawNetworkCallsJSON": req.FailedNetworkCalls, // Always include the raw JSON
		},
		Response: map[string]interface{}{
			"status": "reported",
		},
		RequestHeaders: map[string]string{
			"Content-Type": "multipart/form-data",
		},
		ImageS3URL: imageURL,
	}

	// Log the image URL that will be used
	fmt.Printf("\n=== REPORT HANDLER: TICKET CREATION ===\n")
	fmt.Printf("Image URL being used: %s\n", imageURL)
	if imageURL == "" {
		fmt.Printf("WARNING: Empty image URL will be passed to Jira service\n")
	} else if imageURL == "None" {
		fmt.Printf("WARNING: 'None' literal string will be passed to Jira service\n")
	}
	fmt.Printf("=== END REPORT HANDLER TICKET CREATION ===\n\n")

	response, err := h.jiraService.CreateTicket(c.Request.Context(), ticketReq)
	if err != nil {
		h.logger.Error("Failed to create ticket", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create ticket",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
