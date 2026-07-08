package database

import (
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database/repository"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"gorm.io/gorm"
)

type unitOfWork struct {
	db *gorm.DB
}

// Do implements [port.UnitOfWork].
func (u *unitOfWork) Do(fn func(uow port.UnitOfWork) error) error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		return fn(&unitOfWork{db: tx})
	})
}

// OauthAccountRepo implements [port.UnitOfWork].
func (u *unitOfWork) OauthAccountRepo() port.OauthAccountRepository {
	return repository.NewOauthAccountRepository(u.db)
}

// RoleRepo implements [port.UnitOfWork].
func (u *unitOfWork) RoleRepo() port.RoleRepository {
	return repository.NewRoleRepository(u.db)
}

// SessionRepo implements [port.UnitOfWork].
func (u *unitOfWork) SessionRepo() port.SessionRepository {
	return repository.NewSessionRepository(u.db)
}

// UserRepo implements [port.UnitOfWork].
func (u *unitOfWork) UserRepo() port.UserRepository {
	return repository.NewUserRepository(u.db)
}

func NewUnitOfWork(db *gorm.DB) port.UnitOfWork {
	return &unitOfWork{db: db}
}
