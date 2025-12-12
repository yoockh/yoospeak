package handlers

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yoockh/yoospeak/internal/services"
	"github.com/yoockh/yoospeak/internal/utils"
)

type CVHandler struct {
	svc services.CVFileService
}

func NewCVHandler(svc services.CVFileService) *CVHandler {
	return &CVHandler{svc: svc}
}

func (h *CVHandler) Upload(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		writeError(c, utils.E(utils.CodeInvalidArgument, "CVHandler.Upload", "missing multipart field 'file'", err))
		return
	}

	// basic validation
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".pdf" {
		writeError(c, utils.E(utils.CodeInvalidArgument, "CVHandler.Upload", "only .pdf is allowed", nil))
		return
	}
	if fh.Size <= 0 || fh.Size > 10<<20 {
		writeError(c, utils.E(utils.CodeInvalidArgument, "CVHandler.Upload", "file too large (max 10MB)", nil))
		return
	}

	file, err := fh.Open()
	if err != nil {
		writeError(c, utils.E(utils.CodeInternal, "CVHandler.Upload", "failed to open upload", err))
		return
	}
	defer file.Close()

	// sniff content type (read 512 bytes)
	head := make([]byte, 512)
	n, _ := file.Read(head)
	head = head[:n]
	ct := http.DetectContentType(head)
	if ct != "application/pdf" {
		writeError(c, utils.E(utils.CodeInvalidArgument, "CVHandler.Upload", "invalid content type (must be pdf)", nil))
		return
	}

	// re-compose stream: head + remaining file
	reader := bytes.NewReader(head)
	r := &readJoin{a: reader, b: file}

	objectName := "cv/" + userID + "/" + uuid.NewString() + ".pdf"

	row, err := h.svc.Upload(c.Request.Context(), userID, fh.Filename, int(fh.Size), "application/pdf", objectName, r)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, row)
}

type readJoin struct {
	a *bytes.Reader
	b io.Reader
}

func (r *readJoin) Read(p []byte) (int, error) {
	if r.a != nil && r.a.Len() > 0 {
		return r.a.Read(p)
	}
	return r.b.Read(p)
}
