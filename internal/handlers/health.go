package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/parvez-capri/ronnin/internal/models"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Services  map[string]string `json:"services"`
	Timestamp int64             `json:"timestamp"`
}

// HealthCheckGin godoc
// @Summary      Health check endpoint
// @Description  Get the status of server and its dependencies
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.HealthResponse
// @Failure      503  {object}  models.ErrorResponse
// @Router       /health [get]
func HealthCheckGin(c *gin.Context) {
	health := models.HealthResponse{
		Status: "ok",
		Services: map[string]string{
			"jira": "ok",
			// Add other services here
		},
		Timestamp: time.Now().Unix(),
	}
	c.JSON(200, health)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func checkJiraHealth() string {
	// Implementation of checkJiraHealth function
	return "ok"
}
