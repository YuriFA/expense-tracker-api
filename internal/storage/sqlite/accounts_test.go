package sqlite_test

import (
	"errors"
	"testing"
	"time"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
)

func TestCreateAccount(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cases := map[string]struct {
		name           string
		openingBalance float64
		respError      bool
	}{
		"success": {
			name:           "Account1",
			openingBalance: 100.0,
			respError:      false,
		},
		"negative opening balance": {
			name:           "Account2",
			openingBalance: -100.0,
			respError:      false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount(tc.name, tc.openingBalance)
			if err != nil && !tc.respError {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && tc.respError {
				t.Fatalf("expected error, got nil")
			}

			if account.Name != tc.name || account.OpeningBalance != tc.openingBalance {
				t.Errorf("unexpected account data: got %v", account)
			}

			if err := uuid.Validate(account.Id); err != nil {
				t.Errorf("expected account ID to be set, got empty string")
			}

			if account.ManualAdjustment != 0.0 {
				t.Errorf("expected manual adjustment to be 0.0, got %v", account.ManualAdjustment)
			}

			createdAt, err := time.Parse(time.RFC3339, account.CreatedAt)
			if err != nil {
				t.Errorf("expected created_at to be a valid timestamp, got: %v", account.CreatedAt)
			}

			updatedAt, err := time.Parse(time.RFC3339, account.UpdatedAt)
			if err != nil {
				t.Errorf("expected updated_at to be a valid timestamp, got: %v", account.UpdatedAt)
			}

			if !createdAt.Equal(updatedAt) {
				t.Errorf(
					"expected created_at === updated_at, got created_at: %v, updated_at: %v",
					account.CreatedAt,
					account.UpdatedAt,
				)
			}
		})
	}

	t.Run("non duplicate account ids", func(t *testing.T) {
		account1, err := db.CreateAccount("CreateAccount", 100.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		account2, err := db.CreateAccount("CreateAccount", 200.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if account1.Id == account2.Id {
			t.Errorf(
				"expected different account IDs for duplicate names, got same ID: %v",
				account1.Id,
			)
		}
	})
}

func TestUpdateAccount(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Run("full params updates both params", func(t *testing.T) {
		account, err := db.CreateAccount("Account1", 100.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		time.Sleep(1100 * time.Millisecond) // Ensure updated_at will be different from created_at
		params := storage.UpdateAccountParams{
			Name:             testutil.Ptr("UpdatedAccount"),
			ManualAdjustment: testutil.Ptr(50.0),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updatedAccount.Name != *params.Name || updatedAccount.ManualAdjustment != *params.ManualAdjustment {
			t.Errorf("unexpected account data: got %v", updatedAccount)
		}

		accUpdatedAt, err := time.Parse(time.RFC3339, account.UpdatedAt)
		if err != nil {
			t.Errorf("expected prev updated_at to be a valid timestamp, got: %v", account.UpdatedAt)
		}
		updatedAt, err := time.Parse(time.RFC3339, updatedAccount.UpdatedAt)
		if err != nil {
			t.Errorf("expected next updated_at to be a valid timestamp, got: %v", updatedAccount.UpdatedAt)
		}

		if accUpdatedAt.Equal(updatedAt) {
			t.Errorf(
				"expected updated_at to be different from test account updated_at, got updated_at: %v",
				updatedAccount.UpdatedAt,
			)
		}
	})

	t.Run("only name change", func(t *testing.T) {
		account, err := db.CreateAccount("Account1", 100.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		params := storage.UpdateAccountParams{
			Name: testutil.Ptr("UpdatedAccount"),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updatedAccount.ManualAdjustment != 0.0 {
			t.Errorf(
				"unexpected manual adjustment change: prev %v, got %v",
				account.ManualAdjustment,
				updatedAccount.ManualAdjustment,
			)
		}

		if updatedAccount.Name != *params.Name {
			t.Errorf("unexpected account data: got %v", updatedAccount)
		}
	})

	t.Run("empty params still bumps updated_at", func(t *testing.T) {
		account, err := db.CreateAccount("Account1", 100.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		time.Sleep(1100 * time.Millisecond) // Ensure updated_at will be different from created_at

		updatedAccount, err := db.UpdateAccount(account.Id, storage.UpdateAccountParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updatedAccount.ManualAdjustment != 0.0 {
			t.Errorf(
				"unexpected manual adjustment change: prev %v, got %v",
				account.ManualAdjustment,
				updatedAccount.ManualAdjustment,
			)
		}

		if account.Name != updatedAccount.Name {
			t.Errorf("unexpected account change: prev %v, got %v", account.Name, updatedAccount.Name)
		}

		accUpdatedAt, err := time.Parse(time.RFC3339, account.UpdatedAt)
		if err != nil {
			t.Errorf("expected prev updated_at to be a valid timestamp, got: %v", account.UpdatedAt)
		}
		updatedAt, err := time.Parse(time.RFC3339, updatedAccount.UpdatedAt)
		if err != nil {
			t.Errorf("expected next updated_at to be a valid timestamp, got: %v", updatedAccount.UpdatedAt)
		}

		if accUpdatedAt.Equal(updatedAt) {
			t.Errorf(
				"expected updated_at to be different from test account updated_at, got updated_at: %v",
				updatedAccount.UpdatedAt,
			)
		}
	})

	t.Run("wrong account id return not found", func(t *testing.T) {
		_, err := db.UpdateAccount(uuid.NewString(), storage.UpdateAccountParams{})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, storage.ErrAccountNotFound) {
			t.Fatalf("error mismatch: want `%v`, got: `%v`", storage.ErrAccountNotFound, err)
		}
	})
}

func TestDeleteAccount(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Run("existing account", func(t *testing.T) {
		acc, err := db.CreateAccount("Account1", 100.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = db.DeleteAccount(acc.Id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non existing account", func(t *testing.T) {
		err = db.DeleteAccount(uuid.NewString())
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, storage.ErrAccountNotFound) {
			t.Fatalf("error mismatch: want `%v`, got: `%v`", storage.ErrAccountNotFound, err)
		}
	})

	t.Run("double delete account", func(t *testing.T) {
		acc, err := db.CreateAccount("Account1", 100.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = db.DeleteAccount(acc.Id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = db.DeleteAccount(acc.Id)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, storage.ErrAccountNotFound) {
			t.Errorf("error mismatch: want `%v`, got: `%v`", storage.ErrAccountNotFound, err)
		}
	})
}

func TestGetAccount(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	testAccount, err := db.CreateAccount("Account1", 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := map[string]struct {
		id          string
		respError   bool
		expectedErr error
	}{
		"random non exist uuid": {
			id:          uuid.NewString(),
			respError:   true,
			expectedErr: storage.ErrAccountNotFound,
		},
		"non uuid string": {
			id:          "some id",
			respError:   true,
			expectedErr: storage.ErrAccountNotFound,
		},
		"existing account id": {
			id:        testAccount.Id,
			respError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			account, err := db.GetAccount(tc.id)

			if err != nil && !tc.respError {
				t.Fatalf("unexpected error: %v", err)
			}

			if err != nil && tc.respError {
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("error mismatch: want `%v`, got: `%v`", tc.expectedErr, err)
				}
				// We expect an error and got one, so we can return early to avoid further checks.
				return
			}

			if err == nil && tc.respError {
				t.Fatalf("expected error, got nil")
			}

			if account.Id != tc.id {
				t.Errorf("expected account ID to be %v, got %v", tc.id, account.Id)
			}
		})
	}
}

func createTestAccounts(db *sqlite.Storage) ([]storage.Account, error) {
	testAccounts := []struct {
		name           string
		openingBalance float64
	}{
		{name: "Account1", openingBalance: 100.0},
		{name: "Account2", openingBalance: 200.0},
		{name: "Account3", openingBalance: 300.0},
		{name: "Account4", openingBalance: 400.0},
	}

	result := []storage.Account{}

	for _, acc := range testAccounts {
		account, err := db.CreateAccount(acc.name, acc.openingBalance)
		if err != nil {
			return nil, err
		}

		result = append(result, *account)
	}

	return result, nil
}

func TestGetAccounts(t *testing.T) {
	t.Run("empty accounts in database", func(t *testing.T) {
		db, err := sqlite.New(":memory:")
		if err != nil {
			t.Fatalf("failed to create test database: %v", err)
		}

		accounts, err := db.GetAccounts()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(accounts) != 0 {
			t.Fatalf("expected 0 accounts, got %v", len(accounts))
		}
	})

	t.Run("existing accounts in database", func(t *testing.T) {
		db, err := sqlite.New(":memory:")
		if err != nil {
			t.Fatalf("failed to create test database: %v", err)
		}

		createdAccounts, err := createTestAccounts(db)
		if err != nil {
			t.Fatalf("failed to create test accounts: %v", err)
		}

		accounts, err := db.GetAccounts()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(createdAccounts) != len(accounts) {
			t.Errorf("expected %v accounts, got %v", len(createdAccounts), len(accounts))
		}
	})
}
