package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/christopherdang/vibecloud/api/internal/model"
	"github.com/christopherdang/vibecloud/api/internal/repository"
)

const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I, O, 0, 1 to avoid confusion

type DeviceCodeService struct {
	repo        *repository.DeviceCodeRepository
	authService *AuthService
}

func NewDeviceCodeService(repo *repository.DeviceCodeRepository, authService *AuthService) *DeviceCodeService {
	return &DeviceCodeService{repo: repo, authService: authService}
}

func (s *DeviceCodeService) Generate(ctx context.Context, userID string) (string, error) {
	code, err := generateCode()
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}

	now := time.Now()
	dc := &model.DeviceCode{
		ID:        uuid.New().String(),
		Code:      code,
		UserID:    userID,
		Claimed:   false,
		ExpiresAt: now.Add(5 * time.Minute),
		CreatedAt: now,
	}

	if err := s.repo.Create(ctx, dc); err != nil {
		return "", fmt.Errorf("store device code: %w", err)
	}

	// Opportunistic cleanup of expired codes (fire-and-forget)
	go func() {
		_ = s.repo.DeleteExpired(context.Background())
	}()

	return code, nil
}

func (s *DeviceCodeService) Exchange(ctx context.Context, code string) (*model.TokenPair, error) {
	dc, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("find device code: %w", err)
	}
	if dc == nil {
		return nil, fmt.Errorf("invalid or expired code")
	}

	if err := s.repo.MarkClaimed(ctx, dc.ID); err != nil {
		return nil, fmt.Errorf("mark claimed: %w", err)
	}

	user, err := s.authService.GetUserByID(ctx, dc.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	tokens, err := s.authService.issueTokenPair(user)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	return tokens, nil
}

func generateCode() (string, error) {
	b := make([]byte, 8)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		if err != nil {
			return "", err
		}
		b[i] = codeChars[n.Int64()]
	}
	return string(b), nil
}
