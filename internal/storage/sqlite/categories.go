package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

func (s *Storage) CreateCategory(params storage.CreateCategoryParams) (*storage.Category, error) {
	const op = "storage.sqlite.CreateCategory"

	stmt, err := s.db.Prepare(
		`INSERT INTO categories (id, name, type, icon, color, user_id) VALUES (?, ?, ?, ?, ?, ?) RETURNING id, name, type, icon, color, created_at, updated_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	id := uuid.NewString()
	var category storage.Category
	err = stmt.QueryRow(id, params.Name, params.Type, params.Icon, params.Color, params.UserId).
		Scan(&category.Id, &category.Name, &category.Type, &category.Icon, &category.Color, &category.CreatedAt, &category.UpdatedAt)
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

	setParts, args := newUpdateBuilder().
		addString("name", params.Name).
		addString("type", params.Type).
		addString("icon", params.Icon).
		addString("color", params.Color).
		build(", ")

	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE categories SET %s WHERE id = ? RETURNING id, name, type, icon, color, created_at, updated_at`,
		setParts,
	)

	var category storage.Category
	err := s.db.QueryRow(query, args...).
		Scan(&category.Id, &category.Name, &category.Type, &category.Icon, &category.Color, &category.CreatedAt, &category.UpdatedAt)
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
		if isFKViolationError(err) {
			return fmt.Errorf("%s: %w", op, storage.ErrCategoryHasTransactions)
		}
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
		`SELECT id, name, type, icon, color, created_at, updated_at FROM categories WHERE id = ?`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var category storage.Category
	err = stmt.QueryRow(id).
		Scan(&category.Id, &category.Name, &category.Type, &category.Icon, &category.Color, &category.CreatedAt, &category.UpdatedAt)
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

	query := `SELECT id, name, type, icon, color, created_at, updated_at FROM categories`
	whereParts, args := newWhereBuilder().
		addString("type", params.Type).build(" AND ")

	if len(whereParts) > 0 {
		query = fmt.Sprintf(`%s WHERE %s`, query, whereParts)
	}

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
			&category.Type,
			&category.Icon,
			&category.Color,
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
