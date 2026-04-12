package service

import (
	"context"
	"encoding/json"

	"github.com/yourusername/sub2balance/internal/model"
	"github.com/yourusername/sub2balance/internal/repository"
)

type AuditService struct {
	auditRepo *repository.AuditRepository
}

func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}

func (s *AuditService) Log(ctx context.Context, userID uint, action string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	log := &model.AuditLog{
		UserID:  userID,
		Action:  action,
		Details: string(detailsJSON),
	}

	return s.auditRepo.Create(ctx, log)
}

func (s *AuditService) LogWithRequest(ctx context.Context, userID uint, requestID uint, action string, details interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	log := &model.AuditLog{
		RequestID: &requestID,
		UserID:    userID,
		Action:    action,
		Details:   string(detailsJSON),
	}

	return s.auditRepo.Create(ctx, log)
}

func (s *AuditService) List(ctx context.Context, page, pageSize int) ([]*model.AuditLog, int64, error) {
	offset := (page - 1) * pageSize
	logs, err := s.auditRepo.List(ctx, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.auditRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *AuditService) ListLogs(ctx context.Context, limit, offset int) ([]*model.AuditLog, error) {
	return s.auditRepo.List(ctx, limit, offset)
}
