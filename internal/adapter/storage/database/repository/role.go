package repository

import (
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
)

type roleRepo struct {
	db *database.DB
}

// GetByCode implements [port.RoleRepository].
func (r *roleRepo) GetByCode(code string) (*domain.Role, error) {
	var role domain.Role
	if err := r.db.First(&role, "code = ?", code).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByCodes implements [port.RoleRepository].
func (r *roleRepo) GetByCodes(codes []string) ([]domain.Role, error) {
	var roles []domain.Role
	if err := r.db.Where("code IN ?", codes).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// List implements [port.RoleRepository].
func (r *roleRepo) List() ([]domain.Role, error) {
	var roles []domain.Role
	if err := r.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// NewRoleRepository creates a new role repository instance.
func NewRoleRepository(db *database.DB) port.RoleRepository {
	return &roleRepo{db: db}
}
