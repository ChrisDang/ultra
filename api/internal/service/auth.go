package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/christopherdang/vibecloud/api/internal/model"
	"github.com/christopherdang/vibecloud/api/internal/repository"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	bcryptCost      = 12
	minPasswordLen  = 8
)

type AuthService struct {
	userRepo      *repository.UserRepository
	signingSecret []byte
}

func NewAuthService(userRepo *repository.UserRepository, secret string) *AuthService {
	return &AuthService{userRepo: userRepo, signingSecret: []byte(secret)}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, *model.TokenPair, error) {
	if len(password) < minPasswordLen {
		return nil, nil, fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, model.ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	user := &model.User{
		ID:             uuid.New().String(),
		Email:          email,
		HashedPassword: string(hash),
		Tier:           "free",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	tokens, err := s.issueTokenPair(user)
	return user, tokens, err
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, *model.TokenPair, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, model.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		return nil, nil, model.ErrInvalidCredentials
	}

	tokens, err := s.issueTokenPair(user)
	return user, tokens, err
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.signingSecret, nil
	}, jwt.WithExpirationRequired())

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token")
	}

	sub, _ := claims["sub"].(string)
	user, err := s.userRepo.FindByID(ctx, sub)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueTokenPair(user)
}

func (s *AuthService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

func (s *AuthService) UpdateTier(ctx context.Context, userID, tier string) error {
	return s.userRepo.UpdateTier(ctx, userID, tier)
}

func (s *AuthService) issueTokenPair(user *model.User) (*model.TokenPair, error) {
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"tier":  user.Tier,
		"iat":   now.Unix(),
		"exp":   now.Add(accessTokenTTL).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.signingSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := jwt.MapClaims{
		"sub": user.ID,
		"iat": now.Unix(),
		"exp": now.Add(refreshTokenTTL).Unix(),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.signingSecret)
	if err != nil {
		return nil, err
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}
