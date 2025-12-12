package services

import (
	"context"

	pgrepo "github.com/yoockh/yoospeak/internal/repositories/postgres"
	"github.com/yoockh/yoospeak/internal/utils"
)

type UserService interface {
	HasProfile(ctx context.Context, userID string) (bool, error)
}

type userService struct {
	users pgrepo.UserRepository
}

func NewUserService(users pgrepo.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) HasProfile(ctx context.Context, userID string) (bool, error) {
	const op = "UserService.HasProfile"

	if userID == "" {
		return false, utils.E(utils.CodeInvalidArgument, op, "user_id is required", nil)
	}

	ok, err := s.users.HasProfile(ctx, userID)
	if err != nil {
		return false, utils.E(utils.CodeInternal, op, "failed to check profile existence", err)
	}
	return ok, nil
}
