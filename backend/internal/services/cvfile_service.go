package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yoockh/yoospeak/internal/models"
	pgrepo "github.com/yoockh/yoospeak/internal/repositories/postgres"
	"github.com/yoockh/yoospeak/internal/storage"
	"github.com/yoockh/yoospeak/internal/utils"
)

type CVFileService interface {
	Upload(ctx context.Context, userID string, fileName string, fileSize int, mimeType string, objectName string, r storageReader) (*models.CVFile, error)
}

type storageReader interface {
	Read(p []byte) (n int, err error)
}

type cvFileService struct {
	repo     pgrepo.CVFileRepository
	uploader storage.Uploader
}

func NewCVFileService(repo pgrepo.CVFileRepository, uploader storage.Uploader) CVFileService {
	return &cvFileService{repo: repo, uploader: uploader}
}

func (s *cvFileService) Upload(ctx context.Context, userID string, fileName string, fileSize int, mimeType string, objectName string, r storageReader) (*models.CVFile, error) {
	const op = "CVFileService.Upload"

	if userID == "" || objectName == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "user_id and object_name are required", nil)
	}
	if s.uploader == nil {
		return nil, utils.E(utils.CodeInternal, op, "uploader is not configured", nil)
	}

	storedPath, err := s.uploader.Upload(ctx, objectName, mimeType, r)
	if err != nil {
		return nil, utils.E(utils.CodeUnavailable, op, "failed to upload file", err)
	}

	row := &models.CVFile{
		ID:       uuid.NewString(),
		UserID:   userID,
		FileName: fileName,
		FilePath: storedPath, // <-- was public URL; now object key
		FileSize: fileSize,
		MimeType: mimeType,
		UploadAt: time.Now().UTC(),
	}

	if err := s.repo.Insert(ctx, row); err != nil {
		return nil, utils.E(utils.CodeInternal, op, "failed to persist cv file metadata", err)
	}

	return row, nil
}
