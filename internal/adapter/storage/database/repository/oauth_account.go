package repository

import (
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"gorm.io/gorm"
)

type oauthAccountRepo struct {
	db *gorm.DB
}

// Create implements port.OauthAccountRepository.
func (o *oauthAccountRepo) Create(account *domain.OauthAccount) (*domain.OauthAccount, error) {
	if err := o.db.Create(&account).Error; err != nil {
		return nil, err
	}

	return account, nil
}

// Delete implements port.OauthAccountRepository.
func (o *oauthAccountRepo) Delete(account *domain.OauthAccount) error {
	return o.db.Delete(&account).Error
}

// GetByProvider implements port.OauthAccountRepository.
func (o *oauthAccountRepo) GetByProvider(provider string, providerUserID string) (*domain.OauthAccount, error) {
	var account domain.OauthAccount
	if err := o.db.First(&account, "provider = ? AND provider_user_id = ?", provider, providerUserID).Error; err != nil {
		return nil, err
	}

	return &account, nil
}

// GetByUsername implements port.OauthAccountRepository.
func (o *oauthAccountRepo) GetByUsername(username string) ([]domain.OauthAccount, error) {
	var accounts []domain.OauthAccount
	if err := o.db.Where("username = ?", username).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// NewOauthAccountRepository creates a new oauth account repository instance
func NewOauthAccountRepository(db *gorm.DB) port.OauthAccountRepository {
	return &oauthAccountRepo{db: db}
}
