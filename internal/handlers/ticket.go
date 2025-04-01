package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/parvez-capri/ronnin/internal/errors"
	"github.com/parvez-capri/ronnin/internal/models"
	"github.com/parvez-capri/ronnin/internal/services"
	"go.uber.org/zap"
)

type TicketHandler struct {
	jiraService *services.JiraService
	logger      *zap.Logger
	validate    *validator.Validate
}

func NewTicketHandler(js *services.JiraService, log *zap.Logger, validate *validator.Validate) *TicketHandler {
	return &TicketHandler{
		jiraService: js,
		logger:      log,
		validate:    validate,
	}
}

// CreateTicketGin godoc
// @Summary      Create a new ticket
// @Description  Creates a new JIRA ticket with the provided information
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        request body     models.TicketRequest true "Ticket creation request"
// @Success      201  {object}  models.TicketResponse
// @Failure      400  {object}  models.ErrorResponse "Invalid request body or validation failed"
// @Failure      500  {object}  models.ErrorResponse "Internal server error"
// @Router       /create-ticket [post]
func (h *TicketHandler) CreateTicketGin(c *gin.Context) {
	var req models.TicketRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	response, err := h.jiraService.CreateTicket(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create ticket",
			zap.Error(err),
			zap.String("url", req.URL),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create ticket",
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *TicketHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, errors.NewAPIError(code, message))
}

func (h *TicketHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
