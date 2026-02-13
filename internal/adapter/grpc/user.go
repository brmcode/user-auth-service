package grpc

import (
	"context"
	"errors"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/pb"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserServer struct {
	userRepo port.UserRepository
	cache    port.CacheRepository
	config   *config.Configuration
	pb.UnimplementedUserServiceServer
}

func (u *UserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	user := &domain.User{
		Username:       util.RandomUsername(),
		FirstName:      req.GetFirstName(),
		LastName:       req.GetLastName(),
		Email:          req.GetEmail(),
		HashedPassword: hashedPassword,
		Role:           domain.USER_ROLE,
	}

	createdUser, err := u.userRepo.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, pgErr.Detail)
		}

		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	cacheKey := util.GenerateCacheKey("user", createdUser.Username)
	userSerialized, err := util.Serialize(createdUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cache serialization error: %v", err)
	}

	// Parallel cache operations: set new cache and delete prefix cache concurrently
	errChan := make(chan error, 2)
	go func() {
		errChan <- u.cache.Set(ctx, cacheKey, userSerialized, u.config.Redis.TTL)
	}()
	go func() {
		errChan <- u.cache.DeleteByPrefix(ctx, "users:*")
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return nil, status.Errorf(codes.Internal, "cache update error: %v", err)
		}
	}

	return &pb.CreateUserResponse{

		Username:          createdUser.Username,
		FirstName:         createdUser.FirstName,
		LastName:          createdUser.LastName,
		Email:             createdUser.Email,
		Role:              createdUser.Role,
		PasswordChangedAt: timestamppb.New(createdUser.PasswordChangedAt),
		CreatedAt:         timestamppb.New(createdUser.CreatedAt),
	}, nil

}

func NewUserServer(userRepo port.UserRepository, cache port.CacheRepository, config *config.Configuration) *UserServer {
	return &UserServer{userRepo: userRepo, cache: cache, config: config}
}
