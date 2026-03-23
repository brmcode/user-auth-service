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
	roleRepo port.RoleRepository
	cache    port.CacheRepository
	config   *config.Configuration
	pb.UnimplementedUserServiceServer
}

func (u *UserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	roleCodes := req.GetRoles()
	if len(roleCodes) == 0 {
		roleCodes = []string{domain.USER_ROLE}
	}
	roles, err := u.roleRepo.GetByCodes(roleCodes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load roles: %v", err)
	}
	if len(roles) != len(roleCodes) {
		return nil, status.Error(codes.InvalidArgument, "one or more role codes are invalid")
	}

	user := &domain.User{
		Username:       util.RandomUsername(),
		FirstName:      req.GetFirstName(),
		LastName:       req.GetLastName(),
		Email:          req.GetEmail(),
		HashedPassword: hashedPassword,
		Roles:          roles,
	}

	created, err := u.userRepo.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, pgErr.Detail)
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	key := util.GenerateCacheKey("user", created.Username)
	serialized, err := util.Serialize(created)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cache serialization error: %v", err)
	}
	errChan := make(chan error, 2)
	go func() { errChan <- u.cache.Set(ctx, key, serialized, u.config.Redis.TTL) }()
	go func() { errChan <- u.cache.DeleteByPrefix(ctx, "users:*") }()
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return nil, status.Errorf(codes.Internal, "cache update error: %v", err)
		}
	}

	return &pb.CreateUserResponse{
		Username:          created.Username,
		FirstName:         created.FirstName,
		LastName:          created.LastName,
		Email:             created.Email,
		Roles:             created.RoleCodes(),
		PasswordChangedAt: timestamppb.New(created.PasswordChangedAt),
		CreatedAt:         timestamppb.New(created.CreatedAt),
	}, nil
}

func NewUserServer(
	userRepo port.UserRepository,
	roleRepo port.RoleRepository,
	cache port.CacheRepository,
	cfg *config.Configuration,
) *UserServer {
	return &UserServer{
		userRepo: userRepo,
		roleRepo: roleRepo,
		cache:    cache,
		config:   cfg,
	}
}
