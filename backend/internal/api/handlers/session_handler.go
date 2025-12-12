package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yoockh/yoospeak/internal/models"
	"github.com/yoockh/yoospeak/internal/services"
	"github.com/yoockh/yoospeak/internal/utils"
)

type SessionHandler struct {
	svc services.SessionService
}

func NewSessionHandler(svc services.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

type StartSessionRequest struct {
	Type     string                 `json:"type" binding:"required"`     // interview|casual
	Language string                 `json:"language" binding:"required"` // id|en
	Metadata models.SessionMetadata `json:"metadata"`
}

type StartSessionResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func (h *SessionHandler) Start(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, utils.E(utils.CodeInvalidArgument, "SessionHandler.Start", "invalid request body", err))
		return
	}

	sess, err := h.svc.Start(c.Request.Context(), userID, req.Type, req.Language, req.Metadata)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, StartSessionResponse{
		SessionID: sess.SessionID,
		Status:    sess.Status,
		CreatedAt: sess.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *SessionHandler) Get(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	sessionID := c.Param("session_id")
	sess, err := h.svc.Get(c.Request.Context(), sessionID)
	if err != nil {
		writeError(c, err)
		return
	}

	// basic authorization
	if sess.UserID != userID {
		writeError(c, utils.E(utils.CodeForbidden, "SessionHandler.Get", "forbidden", nil))
		return
	}

	c.JSON(http.StatusOK, sess)
}

func (h *SessionHandler) End(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	sessionID := c.Param("session_id")

	// authorize against existing session
	sess, err := h.svc.Get(c.Request.Context(), sessionID)
	if err != nil {
		writeError(c, err)
		return
	}
	if sess.UserID != userID {
		writeError(c, utils.E(utils.CodeForbidden, "SessionHandler.End", "forbidden", nil))
		return
	}

	ended, err := h.svc.End(c.Request.Context(), sessionID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, ended)
}
