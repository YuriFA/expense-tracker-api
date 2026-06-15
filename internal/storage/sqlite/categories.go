package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

func (s *Storage) CreateCategory(params storage.CreateCategoryParams) (*storage.Category, error) {
	const op = "storage.sqlite.CreateCategory"

	stmt, err := s.db.Prepare(
		`INSERT INTO categories (id, name, type, icon, color, is_default) VALUES (?, ?, ?, ?, ?, ?) RETURNING id, name, type, icon, color, is_default, created_at, updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	id := uuid.NewString()
	var category storage.Category
	err = stmt.QueryRow(id, params.Name, params.Type, params.Icon, params.Color, params.IsDefault).
		Scan(&category.Id, &category.Name, &category.Type, &category.Icon, &category.Color, &category.IsDefault, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &category, nil
}

func (s *Storage) UpdateCategory(
	id string,
	params storage.UpdateCategoryParams,
) (*storage.Category, error) {
	const op = "storage.sqlite.UpdateCategory"

	setParts := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	if params.Name != nil {
		setParts = append(setParts, "name = ?")
		args = append(args, *params.Name)
	}
	if params.Type != nil {
		setParts = append(setParts, "type = ?")
		args = append(args, *params.Type)
	}
	if params.Icon != nil {
		setParts = append(setParts, "icon = ?")
		args = append(args, *params.Icon)
	}
	if params.Color != nil {
		setParts = append(setParts, "color = ?")
		args = append(args, *params.Color)
	}

	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE categories SET %s WHERE id = ? RETURNING id, name, slug, type, icon, color, is_default, created_at, updated_at`,
		strings.Join(setParts, ", "),
	)

	var category storage.Category
	err := s.db.QueryRow(query, args...).
		Scan(&category.Id, &category.Name, &category.Slug, &category.Type, &category.Icon, &category.Color, &category.IsDefault, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrCategoryNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &category, nil
}

func (s *Storage) DeleteCategory(id string) error {
	const op = "storage.sqlite.DeleteCategory"

	stmt, err := s.db.Prepare(
		`DELETE FROM categories WHERE id = ?`,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrCategoryNotFound)
	}

	return nil
}

func (s *Storage) GetCategory(id string) (*storage.Category, error) {
	const op = "storage.sqlite.GetCategory"

	stmt, err := s.db.Prepare(
		`SELECT id, name, slug, type, icon, color, is_default, created_at, updated_at FROM categories WHERE id = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var category storage.Category
	err = stmt.QueryRow(id).
		Scan(&category.Id, &category.Name, &category.Slug, &category.Type, &category.Icon, &category.Color, &category.IsDefault, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrCategoryNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &category, nil
}

func (s *Storage) GetCategories(params storage.GetCategoriesParams) ([]storage.Category, error) {
	const op = "storage.sqlite.GetCategories"

	whereParts := []string{"1=1"}
	args := []any{}
	if params.Type != nil {
		whereParts = append(whereParts, "type = ?")
		args = append(args, *params.Type)
	}

	query := fmt.Sprintf(
		`SELECT id, name, slug, type, icon, color, is_default, created_at, updated_at FROM categories WHERE %s`,
		strings.Join(whereParts, " AND "),
	)

	categories := []storage.Category{}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		category := storage.Category{}
		err := rows.Scan(
			&category.Id,
			&category.Name,
			&category.Slug,
			&category.Type,
			&category.Icon,
			&category.Color,
			&category.IsDefault,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return categories, nil
}
