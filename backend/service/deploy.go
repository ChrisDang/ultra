package service

import (
	"context"
	"time"

	"github.com/christopherdang/vibecloud/backend/repository"
)

const freeDeployLimit = 15

type DeployService struct {
	deployRepo *repository.DeployLogRepository
}

func NewDeployService(deployRepo *repository.DeployLogRepository) *DeployService {
	return &DeployService{deployRepo: deployRepo}
}

type LimitCheck struct {
	Allowed bool   `json:"allowed"`
	Used    int    `json:"used"`
	Limit   int    `json:"limit"`
	Tier    string `json:"tier"`
}

func (s *DeployService) CheckLimit(ctx context.Context, userID, tier string) (*LimitCheck, error) {
	if tier == "premium" {
		return &LimitCheck{Allowed: true, Used: 0, Limit: -1, Tier: tier}, nil
	}

	// Count deploys in the current calendar month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	used, err := s.deployRepo.CountSince(ctx, userID, startOfMonth)
	if err != nil {
		return nil, err
	}

	return &LimitCheck{
		Allowed: used < freeDeployLimit,
		Used:    used,
		Limit:   freeDeployLimit,
		Tier:    tier,
	}, nil
}

func (s *DeployService) LogDeploy(ctx context.Context, userID, projectName string, providers []string, environment, status string) error {
	return s.deployRepo.Create(ctx, userID, projectName, providers, environment, status)
}
