package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"task-api/internal/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrTaskNotFound = errors.New("task not found")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, email, password_hash, name, created_at, updated_at FROM users WHERE email = $1`
	row := r.pool.QueryRow(ctx, query, email)

	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users SET email = $2, name = $3, updated_at = $4 WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.Name, user.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	query := `
		INSERT INTO tasks (id, user_id, title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		task.ID, task.UserID, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt,
	)
	return err
}

func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	query := `SELECT id, user_id, title, description, status, created_at, updated_at FROM tasks WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)

	var task models.Task
	err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) List(ctx context.Context, userID uuid.UUID, filter models.TaskFilter) ([]*models.Task, int, error) {
	baseQuery := `FROM tasks WHERE user_id = $1`
	args := []any{userID}
	argIndex := 2

	if filter.Status != nil {
		baseQuery += " AND status = $" + strconv.Itoa(argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}
	if filter.Priority != nil {
		baseQuery += " AND priority = $" + strconv.Itoa(argIndex)
		args = append(args, *filter.Priority)
		argIndex++
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	orderBy := " ORDER BY created_at DESC"
	if filter.SortBy != "" {
		orderBy = " ORDER BY " + filter.SortBy
		if filter.SortOrder != "" {
			orderBy += " " + filter.SortOrder
		}
	}

	limit := filter.PageSize
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (filter.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	dataQuery := "SELECT id, user_id, title, description, status, created_at, updated_at " + baseQuery + orderBy + " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, &task)
	}

	return tasks, total, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	query := `
		UPDATE tasks SET title = $2, description = $3, status = $4, updated_at = $5 WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, task.ID, task.Title, task.Description, task.Status, task.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TaskStatus) error {
	query := `UPDATE tasks SET status = $2, updated_at = $3 WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}