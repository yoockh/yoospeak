package services

import (
	"context"
	"errors"
	"time"

	"github.com/yoockh/yoospeak/internal/cache"
	"github.com/yoockh/yoospeak/internal/models"
	pgrepo "github.com/yoockh/yoospeak/internal/repositories/postgres"
	"github.com/yoockh/yoospeak/internal/utils"
)

type ProfileService interface {
	GetMe(ctx context.Context, userID string) (*models.Profile, error)
	Upsert(ctx context.Context, p *models.Profile) error
}

type profileService struct {
	profiles pgrepo.ProfileRepository
	cache    cache.Cache
	cacheTTL time.Duration
}

func NewProfileService(profiles pgrepo.ProfileRepository) ProfileService {
	return &profileService{profiles: profiles}
}

func NewProfileServiceWithCache(profiles pgrepo.ProfileRepository, c cache.Cache, ttl time.Duration) ProfileService {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &profileService{profiles: profiles, cache: c, cacheTTL: ttl}
}

func profileKey(userID string) string { return "profile:" + userID }

func (s *profileService) GetMe(ctx context.Context, userID string) (*models.Profile, error) {
	const op = "ProfileService.GetMe"

	if userID == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "user_id is required", nil)
	}

	// cache hit
	if s.cache != nil {
		var cached models.Profile
		if hit, err := s.cache.GetJSON(ctx, profileKey(userID), &cached); err == nil && hit {
			return &cached, nil
		}
	}

	p, err := s.profiles.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			return nil, utils.E(utils.CodeNotFound, op, "profile not found", err)
		}
		return nil, utils.E(utils.CodeInternal, op, "failed to get profile", err)
	}

	// set cache (best-effort)
	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, profileKey(userID), p, s.cacheTTL)
	}

	return p, nil
}

func (s *profileService) Upsert(ctx context.Context, p *models.Profile) error {
	const op = "ProfileService.Upsert"

	if p == nil || p.UserID == "" {
		return utils.E(utils.CodeInvalidArgument, op, "profile.user_id is required", nil)
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}

	if err := s.profiles.Upsert(ctx, p); err != nil {
		return utils.E(utils.CodeInternal, op, "failed to upsert profile", err)
	}

	// invalidate cache
	if s.cache != nil {
		_ = s.cache.Del(ctx, profileKey(p.UserID))
	}

	return nil
}
