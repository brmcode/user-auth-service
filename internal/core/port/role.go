package port

import "github.com/brmcode/user-auth-service/internal/core/domain"

type RoleRepository interface {
	GetByCode(code string) (*domain.Role, error)
	GetByCodes(codes []string) ([]domain.Role, error)
	List() ([]domain.Role, error)
}
