package grpc

import (
	"context"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/i18n"
	"github.com/brmcode/user-auth-service/pkg/pb"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type AuthServer struct {
	config       *config.Configuration
	userRepo     port.UserRepository
	sessionRepo  port.SessionRepository
	tokenService port.TokenService
	pb.UnimplementedAuthServiceServer
}

func (a *AuthServer) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	var (
		user *domain.User
		err  error
	)

	if req.GetRole() != "" {
		user, err = a.userRepo.GetByEmailAndRole(req.GetEmail(), req.GetRole())
	} else {
		user, err = a.userRepo.GetByEmail(req.GetEmail())
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, i18n.Translate("common.internal_error"))
	}

	if err := util.ComparePassword(req.GetPassword(), user.HashedPassword); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid credentials")
	}

	accessToken, accessPayload, err := a.tokenService.GenerateToken(
		uuid.Nil, user.Username, user.RoleCodes(), a.config.Auth.TokenDuration,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, i18n.Translate("common.internal_error"))
	}

	refreshToken, refreshPayload, err := a.tokenService.GenerateToken(
		uuid.Nil, user.Username, user.RoleCodes(), a.config.Auth.RefreshTokenDuration,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, i18n.Translate("common.internal_error"))
	}

	metadata := extractMetadata(ctx)
	session, err := a.sessionRepo.Create(&domain.Session{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    metadata.UserAgent,
		ClientIP:     metadata.ClientIP,
		ExpiresAt:    refreshPayload.ExpiresAt,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, i18n.Translate("common.internal_error"))
	}

	return &pb.LoginUserResponse{
		User: &pb.User{
			Username:          user.Username,
			FirstName:         user.FirstName,
			LastName:          user.LastName,
			Email:             user.Email,
			Roles:             user.RoleCodes(),
			PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
			CreatedAt:         timestamppb.New(user.CreatedAt),
		},
		SessionId:             session.ID.String(),
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  timestamppb.New(accessPayload.ExpiresAt),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: timestamppb.New(refreshPayload.ExpiresAt),
	}, nil
}

func NewAuthServer(
	cfg *config.Configuration,
	userRepo port.UserRepository,
	sessionRepo port.SessionRepository,
	tokenService port.TokenService,
) *AuthServer {
	return &AuthServer{
		config:       cfg,
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		tokenService: tokenService,
	}
}
