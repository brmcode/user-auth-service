package repository

import (
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
)

type userRepo struct {
	db *database.DB
}

// Create implements UserRepository.
func (u *userRepo) Create(user *domain.User) (*domain.User, error) {
	if err := u.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// Delete implements UserRepository.
func (u *userRepo) Delete(user *domain.User) error {
	return u.db.Delete(&user).Error
}

// GetByEmailAndRole implements UserRepository.
func (u *userRepo) GetByEmailAndRole(email string, role string) (*domain.User, error) {
	var user domain.User
	if err := u.db.First(&user, "email = ? AND role = ?", email, role).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByUsername implements UserRepository.
func (u *userRepo) Get(username string) (*domain.User, error) {
	var user domain.User
	if err := u.db.First(&user, "username = ?", username).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Update implements UserRepository.
func (u *userRepo) Update(user *domain.User) (*domain.User, error) {
	if err := u.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *database.DB) port.UserRepository {
	return &userRepo{db: db}
}
