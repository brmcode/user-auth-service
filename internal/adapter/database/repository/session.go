package repository

import (
	"github.com/brmcode/user-auth-service/internal/adapter/database"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/google/uuid"
)

type sessionRepo struct {
	db *database.DB
}

// Create implements SessionRepository.
func (s *sessionRepo) Create(session *domain.Session) (*domain.Session, error) {
	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}

	return session, nil
}

// Get implements SessionRepository.
func (s *sessionRepo) Get(id uuid.UUID) (*domain.Session, error) {
	var session domain.Session
	if err := s.db.First(&session, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func NewSessionRepository(db *database.DB) port.SessionRepository {
	return &sessionRepo{db: db}
}
