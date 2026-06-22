package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
)

func createAccountAndCategory(
	t *testing.T,
	db *sqlite.Storage,
) (storage.Account, storage.Category) {
	t.Helper()

	account, err := db.CreateAccount("Account1", 1000)
	testutil.AssertNoError(t, err)
	category, err := db.CreateCategory(storage.CreateCategoryParams{
		Name:      "Category1",
		Type:      "income",
		Icon:      "icon2",
		Color:     "blue",
		IsDefault: false,
	})
	testutil.AssertNoError(t, err)

	return *account, *category
}

func TestCreateTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	account, category := createAccountAndCategory(t, db)

	cases := map[string]struct {
		params    storage.CreateTransactionParams
		respError bool
	}{
		"with existing category and account": {
			params: storage.CreateTransactionParams{
				Type:        "income",
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  "2024-06-01T00:00:00Z",
				AccountId:   account.Id,
				CategoryId:  category.Id,
			},
			respError: false,
		},
		"with existing category and account but category type not same": {
			params: storage.CreateTransactionParams{
				Type:        "expense",
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  "2024-06-01T00:00:00Z",
				AccountId:   account.Id,
				CategoryId:  category.Id,
			},
			respError: true,
		},
		"with non existing category and account": {
			params: storage.CreateTransactionParams{
				Type:        "income",
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  "2024-06-01T00:00:00Z",
				AccountId:   uuid.NewString(),
				CategoryId:  uuid.NewString(),
			},
			respError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			transaction, err := db.CreateTransaction(tc.params)
			if tc.respError {
				testutil.AssertError(t, err)
				return
			}

			testutil.AssertNoError(t, err)

			testutil.AssertEqual(t, tc.params.Type, transaction.Type)
			testutil.AssertEqual(t, tc.params.Amount, transaction.Amount)
			testutil.AssertEqual(t, tc.params.Description, transaction.Description)
			testutil.AssertEqual(t, tc.params.OccurredAt, transaction.OccurredAt)
			testutil.AssertEqual(t, tc.params.AccountId, transaction.AccountId)
			testutil.AssertEqual(t, tc.params.CategoryId, transaction.CategoryId)

			testutil.AssertValidUUID(t, transaction.Id)

			createdAt := testutil.ParseDatetime(t, transaction.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, transaction.UpdatedAt)
			testutil.AssertEqual(t, createdAt, updatedAt)
		})
	}
}

func TestUpdateTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("full params updates", func(t *testing.T) {
		account, category := createAccountAndCategory(t, db)
		expenseCategory, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category2",
			Type:      "expense",
			Icon:      "icon3",
			Color:     "red",
			IsDefault: false,
		})
		testutil.AssertNoError(t, err)
		transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
			Type:        "income",
			Amount:      1000,
			Description: "Salary1",
			OccurredAt:  "2024-06-01T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  category.Id,
		})
		testutil.AssertNoError(t, err)
		params := storage.UpdateTransactionParams{
			Type:        new("expense"),
			Amount:      new(2000.0),
			Description: new("Updated Salary"),
			OccurredAt:  new("2024-06-02T00:00:00Z"),
			AccountId:   new(account.Id),
			CategoryId:  new(expenseCategory.Id),
		}

		updatedTransaction, err := db.UpdateTransaction(transaction.Id, params)
		testutil.AssertNoError(t, err)

		testutil.AssertEqual(t, *params.Type, updatedTransaction.Type)
		testutil.AssertEqual(t, *params.Amount, updatedTransaction.Amount)
		testutil.AssertEqual(t, *params.Description, updatedTransaction.Description)
		testutil.AssertEqual(t, *params.OccurredAt, updatedTransaction.OccurredAt)
		testutil.AssertEqual(t, *params.AccountId, updatedTransaction.AccountId)
		testutil.AssertEqual(t, *params.CategoryId, updatedTransaction.CategoryId)
	})

	t.Run("only type change", func(t *testing.T) {
		account, category := createAccountAndCategory(t, db)
		transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
			Type:        "income",
			Amount:      1000,
			Description: "Salary1",
			OccurredAt:  "2024-06-01T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  category.Id,
		})
		testutil.AssertNoError(t, err)
		params := storage.UpdateTransactionParams{
			Type: new("expense"),
		}

		_, err = db.UpdateTransaction(transaction.Id, params)
		testutil.AssertErrorIs(t, err, storage.ErrCategoryTypeMismatch)
	})

	t.Run("wrong transaction id return not found", func(t *testing.T) {
		_, err := db.UpdateTransaction(uuid.NewString(), storage.UpdateTransactionParams{})
		testutil.AssertErrorIs(t, err, storage.ErrTransactionNotFound)
	})
}

func TestDeleteTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("existing transaction", func(t *testing.T) {
		account, category := createAccountAndCategory(t, db)
		transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
			Type:        "income",
			Amount:      1000,
			Description: "Salary1",
			OccurredAt:  "2024-06-01T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  category.Id,
		})
		testutil.AssertNoError(t, err)

		err = db.DeleteTransaction(transaction.Id)
		testutil.AssertNoError(t, err)
	})

	t.Run("non existing transaction", func(t *testing.T) {
		err := db.DeleteTransaction(uuid.NewString())
		testutil.AssertErrorIs(t, err, storage.ErrTransactionNotFound)
	})

	t.Run("double delete transaction", func(t *testing.T) {
		account, category := createAccountAndCategory(t, db)
		transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
			Type:        "income",
			Amount:      1000,
			Description: "Salary1",
			OccurredAt:  "2024-06-01T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  category.Id,
		})
		testutil.AssertNoError(t, err)
		err = db.DeleteTransaction(transaction.Id)
		testutil.AssertNoError(t, err)
		err = db.DeleteTransaction(transaction.Id)
		testutil.AssertErrorIs(t, err, storage.ErrTransactionNotFound)
	})
}

func TestGetTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	account, category := createAccountAndCategory(t, db)
	testTransaction, err := db.CreateTransaction(storage.CreateTransactionParams{
		Type:        "income",
		Amount:      1000,
		Description: "Salary1",
		OccurredAt:  "2024-06-01T00:00:00Z",
		AccountId:   account.Id,
		CategoryId:  category.Id,
	})
	testutil.AssertNoError(t, err)

	cases := map[string]struct {
		id          string
		respError   bool
		expectedErr error
	}{
		"random non exist uuid": {
			id:          uuid.NewString(),
			respError:   true,
			expectedErr: storage.ErrTransactionNotFound,
		},
		"non uuid string": {
			id:          "some id",
			respError:   true,
			expectedErr: storage.ErrTransactionNotFound,
		},
		"existing transaction id": {
			id:        testTransaction.Id,
			respError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			transaction, err := db.GetTransaction(tc.id)

			if tc.respError {
				testutil.AssertErrorIs(t, err, tc.expectedErr)
				return
			}

			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, tc.id, transaction.Id)
		})
	}
}

func createTestTransactions(db *sqlite.Storage) ([]storage.Transaction, error) {
	account, err := db.CreateAccount("Account1", 1000)
	if err != nil {
		return nil, err
	}
	incomeCategory, err := db.CreateCategory(storage.CreateCategoryParams{
		Name:      "Category1",
		Type:      "income",
		Icon:      "icon2",
		Color:     "blue",
		IsDefault: false,
	})
	if err != nil {
		return nil, err
	}
	expenseCategory, err := db.CreateCategory(storage.CreateCategoryParams{
		Name:      "Category2",
		Type:      "expense",
		Icon:      "icon3",
		Color:     "red",
		IsDefault: false,
	})
	if err != nil {
		return nil, err
	}

	transactionCreationParams := []storage.CreateTransactionParams{
		{
			Type:        "income",
			Amount:      1000,
			Description: "Salary1",
			OccurredAt:  "2024-06-01T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  incomeCategory.Id,
		},
		{
			Type:        "expense",
			Amount:      2000,
			Description: "Shopping1",
			OccurredAt:  "2024-07-01T14:30:00Z",
			AccountId:   account.Id,
			CategoryId:  expenseCategory.Id,
		},
		{
			Type:        "income",
			Amount:      5000,
			Description: "Salary2",
			OccurredAt:  "2024-05-01T23:59:00Z",
			AccountId:   account.Id,
			CategoryId:  incomeCategory.Id,
		},
		{
			Type:        "expense",
			Amount:      3000,
			Description: "Shopping2",
			OccurredAt:  "2024-07-01T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  expenseCategory.Id,
		},
		{
			Type:        "income",
			Amount:      1000,
			Description: "Salary3",
			OccurredAt:  "2024-05-02T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  incomeCategory.Id,
		},
		{
			Type:        "expense",
			Amount:      1000,
			Description: "Game1",
			OccurredAt:  "2024-05-02T00:00:00Z",
			AccountId:   account.Id,
			CategoryId:  expenseCategory.Id,
		},
	}

	result := []storage.Transaction{}

	for _, params := range transactionCreationParams {
		transaction, err := db.CreateTransaction(params)
		if err != nil {
			return nil, err
		}

		result = append(result, *transaction)
	}

	return result, nil
}

func TestGetTransactions(t *testing.T) {
	t.Run("empty transactions in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		transactions, err := db.GetTransactions(storage.GetTransactionsParams{})
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, 0, len(transactions))
	})

	t.Run("no params", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(db)
		testutil.AssertNoError(t, err)

		transactions, err := db.GetTransactions(storage.GetTransactionsParams{})
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(createdTransactions), len(transactions))
	})

	t.Run("type param = income", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(db)
		testutil.AssertNoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{Type: new("income")},
		)
		testutil.AssertNoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(
			expected,
			func(c storage.Transaction) bool {
				return c.Type == "income"
			},
		)
		testutil.AssertDeepEqual(t, expected, transactions)
	})

	t.Run("sort param occurred_at DESC", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(db)
		testutil.AssertNoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{Sort: new(storage.OccurredAtDesc)},
		)
		testutil.AssertNoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		testutil.AssertDeepEqual(t, expected, transactions)
	})

	t.Run("from date and to date", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(db)
		testutil.AssertNoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{
				FromDate: testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				ToDate:   testutil.GetTimeFromStr(t, "2024-07-01T00:00:00Z"),
			},
		)
		testutil.AssertNoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(expected, func(c storage.Transaction) bool {
			return c.OccurredAt >= "2024-06-01T00:00:00Z" && c.OccurredAt <= "2024-07-01T00:00:00Z"
		})
		testutil.AssertDeepEqual(t, expected, transactions)
	})

	t.Run("from date only", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(db)
		testutil.AssertNoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{
				FromDate: testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
			},
		)
		testutil.AssertNoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(expected, func(c storage.Transaction) bool {
			return c.OccurredAt >= "2024-06-01T00:00:00Z"
		})
		testutil.AssertDeepEqual(t, expected, transactions)
	})

	t.Run("to date only", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(db)
		testutil.AssertNoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{
				ToDate: testutil.GetTimeFromStr(t, "2024-07-01T00:00:00Z"),
			},
		)
		testutil.AssertNoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(expected, func(c storage.Transaction) bool {
			return c.OccurredAt <= "2024-07-01T00:00:00Z"
		})
		testutil.AssertDeepEqual(t, expected, transactions)
	})

	t.Run("limit = 2", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(db)
		testutil.AssertNoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{Limit: new(2)},
		)
		testutil.AssertNoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		testutil.AssertDeepEqual(t, expected[0:2], transactions)
	})
}
