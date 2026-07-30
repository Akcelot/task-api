package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"task-api/internal/models"
	"task-api/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrUserNotFound       = errors.New("user not found")
	ErrTaskNotFound       = errors.New("task not found")
)

type UserService struct {
	repo            *repository.UserRepository
	secretKey       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewUserService(repo *repository.UserRepository, secretKey string, accessTTL, refreshTTL time.Duration) *UserService {
	return &UserService{
		repo:            repo,
		secretKey:       secretKey,
		accessTokenTTL:   accessTTL,
		refreshTokenTTL:  refreshTTL,
	}
}

func (s *UserService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	existing, _ := s.repo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.generateTokens(user)
}

func (s *UserService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokens(user)
}

func (s *UserService) Refresh(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return s.generateTokens(user)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	return s.repo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) ValidateAccessToken(tokenString string) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *UserService) generateTokens(user *models.User) (*models.AuthResponse, error) {
	now := time.Now()

	accessClaims := &models.Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}

	refreshClaims := &models.Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)

	accessString, err := accessToken.SignedString([]byte(s.secretKey))
	if err != nil {
		return nil, err
	}

	refreshString, err := refreshToken.SignedString([]byte(s.secretKey))
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessString,
		RefreshToken: refreshString,
		User:         *user,
	}, nil
}

type TaskService struct {
	repo *repository.TaskRepository
}

func NewTaskService(repo *repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) Create(ctx context.Context, userID uuid.UUID, req models.CreateTaskRequest) (*models.Task, error) {
	now := time.Now()
	status := req.Status
	if status == "" {
		status = models.TaskStatusPending
	}
	priority := req.Priority
	if priority == "" {
		priority = models.PriorityMedium
	}

	task := &models.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     req.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

func (s *TaskService) List(ctx context.Context, userID uuid.UUID, filter models.TaskFilter) ([]*models.Task, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.Offset = (filter.Page - 1) * filter.PageSize
	filter.Limit = filter.PageSize

	return s.repo.List(ctx, userID, filter)
}

func (s *TaskService) Update(ctx context.Context, task *models.Task) error {
	task.UpdatedAt = time.Now()
	return s.repo.Update(ctx, task)
}

func (s *TaskService) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TaskStatus) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *TaskService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

type AuthService struct {
	userService *UserService
}

func NewAuthService(userService *UserService) *AuthService {
	return &AuthService{userService: userService}
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	return s.userService.Register(ctx, req)
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	return s.userService.Login(ctx, req)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	return s.userService.Refresh(ctx, refreshToken)
}

func (s *AuthService) ValidateToken(tokenString string) (*models.Claims, error) {
	return s.userService.ValidateAccessToken(tokenString)
}