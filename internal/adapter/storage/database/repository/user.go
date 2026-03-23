package repository

import (
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"gorm.io/gorm"
)

type userRepo struct {
	db *database.DB
}

// Create implements [port.UserRepository].
func (u *userRepo) Create(user *domain.User) (*domain.User, error) {
	err := u.db.DB.Transaction(func(tx *gorm.DB) error {
		roles := user.Roles
		user.Roles = nil

		if err := tx.Create(user).Error; err != nil {
			return err
		}

		if len(roles) > 0 {
			if err := tx.Model(user).
				Association("Roles").
				Replace(roles); err != nil {
				return err
			}
			user.Roles = roles
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil

}

// Get implements [port.UserRepository].
func (u *userRepo) Get(username string) (*domain.User, error) {
	var user domain.User
	if err := u.db.Preload("Roles").
		First(&user, "username = ?", username).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail implements [port.UserRepository].
func (u *userRepo) GetByEmail(email string) (*domain.User, error) {
	var user domain.User
	if err := u.db.Preload("Roles").
		First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmailAndRole implements [port.UserRepository].
func (u *userRepo) GetByEmailAndRole(email string, roleCode string) (*domain.User, error) {
	var user domain.User
	err := u.db.Preload("Roles").
		Joins("JOIN user_roles ON user_roles.username = users.username").
		First(&user, "users.email = ? AND user_roles.role_code = ?", email, roleCode).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmailUnscoped implements [port.UserRepository].
func (u *userRepo) GetByEmailUnscoped(email string) (*domain.User, error) {
	var user domain.User
	if err := u.db.Preload("Roles").
		Unscoped().
		First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Update implements [port.UserRepository].
func (u *userRepo) Update(user *domain.User) (*domain.User, error) {
	err := u.db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(user).Error; err != nil {
			return err
		}
		return tx.Model(user).Association("Roles").Replace(user.Roles)

	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Delete implements [port.UserRepository].
func (u *userRepo) Delete(user *domain.User) error {
	return u.db.Delete(&user).Error
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *database.DB) port.UserRepository {
	return &userRepo{db: db}
}
