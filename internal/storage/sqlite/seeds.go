package sqlite

import (
	"fmt"

	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
)

func (s *Storage) SeedCategories() error {
	const op = "storage.sqlite.SeedCategories"

	seedCategories := []storage.CreateDefaultCategoryParams{
		{
			Name:  "Food",
			Slug:  "food",
			Type:  "expense",
			Icon:  "🍔",
			Color: "#FF6347",
		},
		{
			Name:  "Transport",
			Slug:  "transport",
			Type:  "expense",
			Icon:  "🚗",
			Color: "#1E90FF",
		},
		{
			Name:  "Entertainment",
			Slug:  "entertainment",
			Type:  "expense",
			Icon:  "🎬",
			Color: "#FFD700",
		},
		{
			Name:  "Salary",
			Slug:  "salary",
			Type:  "income",
			Icon:  "💼",
			Color: "#32CD32",
		},
		{
			Name:  "Freelance",
			Slug:  "freelance",
			Type:  "income",
			Icon:  "🖥️",
			Color: "#8A2BE2",
		},
		{
			Name:  "Health",
			Slug:  "health",
			Type:  "expense",
			Icon:  "💊",
			Color: "#FF69B4",
		},
		{
			Name:  "Education",
			Slug:  "education",
			Type:  "expense",
			Icon:  "📚",
			Color: "#20B2AA",
		},
		{
			Name:  "Investment",
			Slug:  "investment",
			Type:  "income",
			Icon:  "📈",
			Color: "#FF4500",
		},
		{
			Name:  "Gifts",
			Slug:  "gifts",
			Type:  "income",
			Icon:  "🎁",
			Color: "#FF1493",
		},
		{
			Name:  "Utilities",
			Slug:  "utilities",
			Type:  "expense",
			Icon:  "💡",
			Color: "#00CED1",
		},
		{
			Name:  "Travel",
			Slug:  "travel",
			Type:  "expense",
			Icon:  "✈️",
			Color: "#FF8C00",
		},
		{
			Name:  "Miscellaneous",
			Slug:  "miscellaneous",
			Type:  "expense",
			Icon:  "🛍️",
			Color: "#A52A2A",
		},
		{
			Name:  "Bonus",
			Slug:  "bonus",
			Type:  "income",
			Icon:  "🎉",
			Color: "#32CD32",
		},
		{
			Name:  "Rent",
			Slug:  "rent",
			Type:  "expense",
			Icon:  "🏠",
			Color: "#8B4513",
		},
		{
			Name:  "Savings",
			Slug:  "savings",
			Type:  "income",
			Icon:  "💰",
			Color: "#228B22",
		},
		{
			Name:  "Charity",
			Slug:  "charity",
			Type:  "expense",
			Icon:  "❤️",
			Color: "#FF69B4",
		},
		{
			Name:  "Side Hustle",
			Slug:  "side-hustle",
			Type:  "income",
			Icon:  "🛠️",
			Color: "#8A2BE2",
		},
		{
			Name:  "Subscriptions",
			Slug:  "subscriptions",
			Type:  "expense",
			Icon:  "📱",
			Color: "#1E90FF",
		},
		{
			Name:  "Other Income",
			Slug:  "other-income",
			Type:  "income",
			Icon:  "💵",
			Color: "#32CD32",
		},
		{
			Name:  "Other Expense",
			Slug:  "other-expense",
			Type:  "expense",
			Icon:  "🛒",
			Color: "#A52A2A",
		},
		{
			Name:  "Health Insurance",
			Slug:  "health-insurance",
			Type:  "expense",
			Icon:  "🏥",
			Color: "#FF69B4",
		},
		{
			Name:  "Car Maintenance",
			Slug:  "car-maintenance",
			Type:  "expense",
			Icon:  "🔧",
			Color: "#1E90FF",
		},
		{
			Name:  "Grocery",
			Slug:  "grocery",
			Type:  "expense",
			Icon:  "🛒",
			Color: "#FF6347",
		},
		{
			Name:  "Dining Out",
			Slug:  "dining-out",
			Type:  "expense",
			Icon:  "🍽️",
			Color: "#FFD700",
		},
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO categories (id, name, slug, type, icon, color, is_default) VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	for _, category := range seedCategories {
		id := uuid.NewString()
		_, err := stmt.Exec(
			id,
			category.Name,
			category.Slug,
			category.Type,
			category.Icon,
			category.Color,
			true, // Set is_default to true for all seeded categories
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
