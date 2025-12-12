package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yoockh/yoospeak/internal/models"
	"github.com/yoockh/yoospeak/internal/services"
	"github.com/yoockh/yoospeak/internal/utils"
	"gorm.io/datatypes"
)

type ProfileHandler struct {
	svc services.ProfileService
}

func NewProfileHandler(svc services.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

func (h *ProfileHandler) Me(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	p, err := h.svc.GetMe(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, p)
}

type UpdateProfileRequest struct {
	FullName    *string `json:"full_name,omitempty"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	CVText      *string `json:"cv_text,omitempty"`

	Skills *[]string `json:"skills,omitempty"`

	// JSONB fields (raw)
	Experience  *json.RawMessage `json:"experience,omitempty"`
	Education   *json.RawMessage `json:"education,omitempty"`
	Preferences *json.RawMessage `json:"preferences,omitempty"`
}

func (h *ProfileHandler) Update(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, utils.E(utils.CodeInvalidArgument, "ProfileHandler.Update", "invalid request body", err))
		return
	}

	// Load existing (if not found => create new)
	var existing *models.Profile
	existing, err := h.svc.GetMe(c.Request.Context(), userID)
	if err != nil {
		// if profile not found, create blank
		if utils.IsCode(err, utils.CodeNotFound) {
			existing = &models.Profile{UserID: userID}
		} else {
			writeError(c, err)
			return
		}
	}

	// Apply partial updates
	if req.FullName != nil {
		existing.FullName = *req.FullName
	}
	if req.PhoneNumber != nil {
		existing.PhoneNumber = *req.PhoneNumber
	}
	if req.CVText != nil {
		existing.CVText = *req.CVText
	}
	if req.Skills != nil {
		existing.Skills = *req.Skills
	}
	if req.Experience != nil {
		existing.Experience = datatypes.JSON(*req.Experience)
	}
	if req.Education != nil {
		existing.Education = datatypes.JSON(*req.Education)
	}
	if req.Preferences != nil {
		existing.Preferences = datatypes.JSON(*req.Preferences)
	}

	existing.UpdatedAt = time.Now().UTC()

	if err := h.svc.Upsert(c.Request.Context(), existing); err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, existing)
}
