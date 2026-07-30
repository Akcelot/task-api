package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Task struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      TaskStatus `json:"status" db:"status"`
	Priority    Priority   `json:"priority" db:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty" db:"due_date"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type CreateTaskRequest struct {
	Title       string     `json:"title" validate:"required,min=1,max=200"`
	Description string     `json:"description" validate:"max=2000"`
	Status      TaskStatus `json:"status" validate:"omitempty,oneof=pending in_progress done"`
	Priority    Priority   `json:"priority" validate:"omitempty,oneof=low medium high"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type UpdateTaskRequest struct {
	Title       *string     `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string     `json:"description,omitempty" validate:"omitempty,max=2000"`
	Status      *TaskStatus `json:"status,omitempty" validate:"omitempty,oneof=pending in_progress done"`
	Priority    *Priority   `json:"priority,omitempty" validate:"omitempty,oneof=low medium high"`
	DueDate     *time.Time  `json:"due_date,omitempty"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required,min=1,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type TaskListResponse struct {
	Tasks      []Task `json:"tasks"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

type TaskFilter struct {
	Status    *TaskStatus `json:"status,omitempty"`
	Priority  *Priority   `json:"priority,omitempty"`
	Page      int         `json:"page" validate:"omitempty,min=1"`
	PageSize  int         `json:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy    string      `json:"sort_by" validate:"omitempty,oneof=created_at updated_at due_date priority"`
	SortOrder string      `json:"sort_order" validate:"omitempty,oneof=asc desc"`
	Offset    int         `json:"-"`
	Limit     int         `json:"-"`
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}
