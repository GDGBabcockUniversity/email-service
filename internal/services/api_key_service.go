package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type ApiKeyService struct {
	apiKeyRepo  domain.ApiKeyRepository
	projectRepo domain.ProjectRepository
}

func NewApiKeyService(apiKeyRepo domain.ApiKeyRepository, projectRepo domain.ProjectRepository) *ApiKeyService {
	return &ApiKeyService{apiKeyRepo: apiKeyRepo, projectRepo: projectRepo}
}

func (s *ApiKeyService) CreateKey(ctx context.Context, projectID string) (*domain.ApiKey, string, error) {
	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return nil, "", err
	}
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return nil, "", err
	}
	rawKey := "gdg_" + hex.EncodeToString(value)
	key := &domain.ApiKey{Key: hashKey(rawKey), ProjectID: projectID}
	if err := s.apiKeyRepo.Save(ctx, key); err != nil {
		return nil, "", err
	}
	return key, rawKey, nil
}

func (s *ApiKeyService) FindAll(ctx context.Context, projectID string) ([]domain.ApiKey, error) {
	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return nil, err
	}
	return s.apiKeyRepo.FindByProjectID(ctx, projectID)
}

func (s *ApiKeyService) RevokeKey(ctx context.Context, projectID, id string) error {
	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return err
	}
	return s.apiKeyRepo.Revoke(ctx, projectID, id)
}

func (s *ApiKeyService) Authenticate(ctx context.Context, rawKey string) (string, error) {
	if rawKey == "" {
		return "", domain.ErrUnauthorized
	}
	key, err := s.apiKeyRepo.FindByKey(ctx, hashKey(rawKey))
	if errors.Is(err, domain.ErrNotFound) {
		return "", domain.ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	if key.Revoked {
		return "", domain.ErrUnauthorized
	}
	if _, err := s.projectRepo.FindByID(ctx, key.ProjectID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.ErrUnauthorized
		}
		return "", err
	}
	return key.ProjectID, nil
}

func hashKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}
