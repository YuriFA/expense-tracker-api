package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

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
			if tc.respError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tc.name, account.Name)
			require.Equal(t, tc.openingBalance, account.OpeningBalance)

			testutil.AssertValidUUID(t, account.Id)
			assert.Equal(t, 0.0, account.ManualAdjustment)

			createdAt := testutil.ParseDatetime(t, account.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, account.UpdatedAt)
			assert.Equal(t, createdAt, updatedAt)
		})
	}

	t.Run("non duplicate account ids", func(t *testing.T) {
		account1 := seedAccount(t, db, 100.0)
		account2 := seedAccount(t, db, 200.0)
		require.NotEqual(t, account1.Id, account2.Id)
	})
}

func TestUpdateAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("full params updates both params", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		params := storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(50.0),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		require.NoError(t, err)
		require.Equal(t, *params.Name, updatedAccount.Name)
		require.Equal(t, *params.ManualAdjustment, updatedAccount.ManualAdjustment)
	})

	t.Run("only name change", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		params := storage.UpdateAccountParams{
			Name: new("UpdatedAccount"),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		require.NoError(t, err)
		require.Equal(t, 0.0, updatedAccount.ManualAdjustment)
		require.Equal(t, *params.Name, updatedAccount.Name)
	})

	t.Run("wrong account id return not found", func(t *testing.T) {
		_, err := db.UpdateAccount(uuid.NewString(), storage.UpdateAccountParams{})
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestDeleteAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("existing account", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		err := db.DeleteAccount(account.Id)
		require.NoError(t, err)
	})

	t.Run("non existing account", func(t *testing.T) {
		err := db.DeleteAccount(uuid.NewString())
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})

	t.Run("double delete account", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		err := db.DeleteAccount(account.Id)
		require.NoError(t, err)
		err = db.DeleteAccount(account.Id)
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestGetAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	testAccount := seedAccount(t, db, 100.0)

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

			if tc.respError {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.id, account.Id)
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
		db := sqlite.NewTestDB(t)
		accounts, err := db.GetAccounts()
		require.NoError(t, err)
		require.Equal(t, 0, len(accounts))
	})

	t.Run("existing accounts in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		createdAccounts, err := createTestAccounts(db)
		require.NoError(t, err)
		accounts, err := db.GetAccounts()
		require.NoError(t, err)
		require.Equal(t, len(createdAccounts), len(accounts))
	})
}
