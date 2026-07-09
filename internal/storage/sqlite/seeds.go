package sqlite

type CategoryTemplate struct {
	Name  string
	Type  string
	Icon  string
	Color string
}

var defaultCategories = []CategoryTemplate{
	{
		Name:  "Food",
		Type:  "expense",
		Icon:  "🍔",
		Color: "#FF6347",
	},
	{
		Name:  "Transport",
		Type:  "expense",
		Icon:  "🚗",
		Color: "#1E90FF",
	},
	{
		Name:  "Entertainment",
		Type:  "expense",
		Icon:  "🎬",
		Color: "#FFD700",
	},
	{
		Name:  "Salary",
		Type:  "income",
		Icon:  "💼",
		Color: "#32CD32",
	},
	{
		Name:  "Freelance",
		Type:  "income",
		Icon:  "🖥️",
		Color: "#8A2BE2",
	},
	{
		Name:  "Health",
		Type:  "expense",
		Icon:  "💊",
		Color: "#FF69B4",
	},
	{
		Name:  "Education",
		Type:  "expense",
		Icon:  "📚",
		Color: "#20B2AA",
	},
	{
		Name:  "Investment",
		Type:  "income",
		Icon:  "📈",
		Color: "#FF4500",
	},
	{
		Name:  "Gifts",
		Type:  "income",
		Icon:  "🎁",
		Color: "#FF1493",
	},
	{
		Name:  "Utilities",
		Type:  "expense",
		Icon:  "💡",
		Color: "#00CED1",
	},
	{
		Name:  "Travel",
		Type:  "expense",
		Icon:  "✈️",
		Color: "#FF8C00",
	},
	{
		Name:  "Miscellaneous",
		Type:  "expense",
		Icon:  "🛍️",
		Color: "#A52A2A",
	},
	{
		Name:  "Bonus",
		Type:  "income",
		Icon:  "🎉",
		Color: "#32CD32",
	},
	{
		Name:  "Rent",
		Type:  "expense",
		Icon:  "🏠",
		Color: "#8B4513",
	},
	{
		Name:  "Savings",
		Type:  "income",
		Icon:  "💰",
		Color: "#228B22",
	},
	{
		Name:  "Charity",
		Type:  "expense",
		Icon:  "❤️",
		Color: "#FF69B4",
	},
	{
		Name:  "Side Hustle",
		Type:  "income",
		Icon:  "🛠️",
		Color: "#8A2BE2",
	},
	{
		Name:  "Subscriptions",
		Type:  "expense",
		Icon:  "📱",
		Color: "#1E90FF",
	},
	{
		Name:  "Other Income",
		Type:  "income",
		Icon:  "💵",
		Color: "#32CD32",
	},
	{
		Name:  "Other Expense",
		Type:  "expense",
		Icon:  "🛒",
		Color: "#A52A2A",
	},
	{
		Name:  "Health Insurance",
		Type:  "expense",
		Icon:  "🏥",
		Color: "#FF69B4",
	},
	{
		Name:  "Car Maintenance",
		Type:  "expense",
		Icon:  "🔧",
		Color: "#1E90FF",
	},
	{
		Name:  "Grocery",
		Type:  "expense",
		Icon:  "🛒",
		Color: "#FF6347",
	},
	{
		Name:  "Dining Out",
		Type:  "expense",
		Icon:  "🍽️",
		Color: "#FFD700",
	},
}
