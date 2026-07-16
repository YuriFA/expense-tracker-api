package sqlite

import "github.com/yurifa/expense-tracker-api/internal/storage"

type CategoryTemplate struct {
	Name  string
	Type  storage.TransactionType
	Icon  string
	Color string
}

var defaultCategories = []CategoryTemplate{ //nolint:gochecknoglobals // seed data for new users, not runtime state
	{
		Name:  "Food",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🍔",
		Color: "#FF6347",
	},
	{
		Name:  "Transport",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🚗",
		Color: "#1E90FF",
	},
	{
		Name:  "Entertainment",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🎬",
		Color: "#FFD700",
	},
	{
		Name:  "Salary",
		Type:  storage.TransactionTypeIncome,
		Icon:  "💼",
		Color: "#32CD32",
	},
	{
		Name:  "Freelance",
		Type:  storage.TransactionTypeIncome,
		Icon:  "🖥️",
		Color: "#8A2BE2",
	},
	{
		Name:  "Health",
		Type:  storage.TransactionTypeExpense,
		Icon:  "💊",
		Color: "#FF69B4",
	},
	{
		Name:  "Education",
		Type:  storage.TransactionTypeExpense,
		Icon:  "📚",
		Color: "#20B2AA",
	},
	{
		Name:  "Investment",
		Type:  storage.TransactionTypeIncome,
		Icon:  "📈",
		Color: "#FF4500",
	},
	{
		Name:  "Gifts",
		Type:  storage.TransactionTypeIncome,
		Icon:  "🎁",
		Color: "#FF1493",
	},
	{
		Name:  "Utilities",
		Type:  storage.TransactionTypeExpense,
		Icon:  "💡",
		Color: "#00CED1",
	},
	{
		Name:  "Travel",
		Type:  storage.TransactionTypeExpense,
		Icon:  "✈️",
		Color: "#FF8C00",
	},
	{
		Name:  "Miscellaneous",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🛍️",
		Color: "#A52A2A",
	},
	{
		Name:  "Bonus",
		Type:  storage.TransactionTypeIncome,
		Icon:  "🎉",
		Color: "#32CD32",
	},
	{
		Name:  "Rent",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🏠",
		Color: "#8B4513",
	},
	{
		Name:  "Savings",
		Type:  storage.TransactionTypeIncome,
		Icon:  "💰",
		Color: "#228B22",
	},
	{
		Name:  "Charity",
		Type:  storage.TransactionTypeExpense,
		Icon:  "❤️",
		Color: "#FF69B4",
	},
	{
		Name:  "Side Hustle",
		Type:  storage.TransactionTypeIncome,
		Icon:  "🛠️",
		Color: "#8A2BE2",
	},
	{
		Name:  "Subscriptions",
		Type:  storage.TransactionTypeExpense,
		Icon:  "📱",
		Color: "#1E90FF",
	},
	{
		Name:  "Other Income",
		Type:  storage.TransactionTypeIncome,
		Icon:  "💵",
		Color: "#32CD32",
	},
	{
		Name:  "Other Expense",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🛒",
		Color: "#A52A2A",
	},
	{
		Name:  "Health Insurance",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🏥",
		Color: "#FF69B4",
	},
	{
		Name:  "Car Maintenance",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🔧",
		Color: "#1E90FF",
	},
	{
		Name:  "Grocery",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🛒",
		Color: "#FF6347",
	},
	{
		Name:  "Dining Out",
		Type:  storage.TransactionTypeExpense,
		Icon:  "🍽️",
		Color: "#FFD700",
	},
}
