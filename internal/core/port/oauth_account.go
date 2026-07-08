package port

import "github.com/brmcode/user-auth-service/internal/core/domain"

type OauthAccountRepository interface {
	Create(account *domain.OauthAccount) (*domain.OauthAccount, error)
	GetByProvider(provider, providerUserID string) (*domain.OauthAccount, error)
	GetByUsername(username string) ([]domain.OauthAccount, error)
	Delete(account *domain.OauthAccount) error
}
