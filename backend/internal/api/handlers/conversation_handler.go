package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yoockh/yoospeak/internal/services"
)

type ConversationHandler struct {
	svc services.ConversationService
}

func NewConversationHandler(svc services.ConversationService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

type ConversationListResponse struct {
	SessionID     string      `json:"session_id"`
	Conversations interface{} `json:"conversations"`
}

func (h *ConversationHandler) ListBySession(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	sessionID := c.Param("session_id")

	limit := 50
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	rows, err := h.svc.ListBySession(c.Request.Context(), userID, sessionID, limit)
	if err != nil {
		writeError(c, err)
		return
	}

	// Repo returns DESC; frontend biasanya enak ASC
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":    sessionID,
		"conversations": rows,
	})
}
