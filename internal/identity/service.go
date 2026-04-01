package identity

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Bootstrap loads the existing identity or generates and persists a new one.
func (s *Service) Bootstrap(ctx context.Context) (*Identity, error) {
	id, err := s.repo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load identity: %w", err)
	}
	if id != nil {
		return id, nil
	}

	id, err = Generate()
	if err != nil {
		return nil, fmt.Errorf("generate identity: %w", err)
	}
	if err := s.repo.Save(ctx, id); err != nil {
		return nil, fmt.Errorf("save identity: %w", err)
	}
	return id, nil
}
