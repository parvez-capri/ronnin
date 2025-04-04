package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

// GetAllTicketsGin handles GET requests to retrieve all tickets
// @Summary      Get All Tickets
// @Description  Retrieves all tickets from the database
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Success      200  {array}   services.FlattenedTicket
// @Failure      500  {object}  models.ErrorResponse
// @Router       /tickets [get]
func (h *TicketHandler) GetAllTicketsGin(c *gin.Context) {
	if h.jiraService.GetMongoService() == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Database not available",
			Details: "MongoDB service is not configured",
		})
		return
	}

	tickets, err := h.jiraService.GetMongoService().GetAllTickets(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to retrieve tickets", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve tickets",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tickets)
}

// GetTicketByIDGin handles GET requests to retrieve a ticket by ID
// @Summary      Get Ticket by ID
// @Description  Retrieves a ticket by its Jira ID
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id  path      string  true  "Ticket ID"
// @Success      200  {object}  services.FlattenedTicket
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /tickets/{id} [get]
func (h *TicketHandler) GetTicketByIDGin(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request",
			Details: "Ticket ID is required",
		})
		return
	}

	if h.jiraService.GetMongoService() == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Database not available",
			Details: "MongoDB service is not configured",
		})
		return
	}

	ticket, err := h.jiraService.GetMongoService().GetTicketByJiraID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to retrieve ticket", zap.Error(err), zap.String("id", id))

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Ticket not found",
				Details: fmt.Sprintf("Ticket with ID %s not found", id),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to retrieve ticket",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

func (h *TicketHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, errors.NewAPIError(code, message))
}

func (h *TicketHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
