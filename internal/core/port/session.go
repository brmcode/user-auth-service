package port

import (
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/google/uuid"
)

type SessionRepository interface {
	Create(session *domain.Session) (*domain.Session, error)
	Get(id uuid.UUID) (*domain.Session, error)
	GetByToken(token string) (*domain.Session, error)
	Update(session *domain.Session) (*domain.Session, error)
	BlockAllSessions(username string) error
}
