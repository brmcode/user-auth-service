package repository

import (
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"gorm.io/gorm"
)

type userRepo struct {
	db *gorm.DB
}

// Create implements [port.UserRepository].
func (u *userRepo) Create(user *domain.User) (*domain.User, error) {
	roles := user.Roles
	user.Roles = nil
	defer func() {
		user.Roles = roles
	}()

	if err := u.db.Create(user).Error; err != nil {
		return nil, err
	}

	if len(roles) > 0 {
		if err := u.db.Model(user).
			Association("Roles").
			Replace(roles); err != nil {
			return nil, err
		}
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
	if err := u.db.Save(user).Error; err != nil {
		return nil, err
	}
	if err := u.db.Model(user).Association("Roles").Replace(user.Roles); err != nil {
		return nil, err
	}

	return user, nil
}

// Delete implements [port.UserRepository].
func (u *userRepo) Delete(user *domain.User) error {
	return u.db.Delete(&user).Error
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *gorm.DB) port.UserRepository {
	return &userRepo{db: db}
}
