package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
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
			t.Parallel()
			account, err := db.CreateAccount(tc.name, tc.openingBalance)
			if tc.respError {
				testutil.AssertError(t, err)
				return
			}

			testutil.AssertNoError(t, err)

			testutil.AssertEqual(t, account.Name, tc.name)
			testutil.AssertEqual(t, account.OpeningBalance, tc.openingBalance)

			testutil.AssertValidUUID(t, account.Id)
			testutil.AssertEqual(t, account.ManualAdjustment, 0.0)

			createdAt := testutil.ParseDatetime(t, account.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, account.UpdatedAt)
			testutil.AssertEqual(t, updatedAt, createdAt)
		})
	}

	t.Run("non duplicate account ids", func(t *testing.T) {
		account1, err := db.CreateAccount("CreateAccount", 100.0)
		testutil.AssertNoError(t, err)
		account2, err := db.CreateAccount("CreateAccount", 200.0)
		testutil.AssertNoError(t, err)
		testutil.AssertNotEqual(t, account1.Id, account2.Id)
	})
}

func TestUpdateAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("full params updates both params", func(t *testing.T) {
		account, err := db.CreateAccount("Account1", 100.0)
		testutil.AssertNoError(t, err)
		params := storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(50.0),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, updatedAccount.Name, *params.Name)
		testutil.AssertEqual(t, updatedAccount.ManualAdjustment, *params.ManualAdjustment)
	})

	t.Run("only name change", func(t *testing.T) {
		account, err := db.CreateAccount("Account1", 100.0)
		testutil.AssertNoError(t, err)
		params := storage.UpdateAccountParams{
			Name: new("UpdatedAccount"),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, updatedAccount.ManualAdjustment, 0.0)
		testutil.AssertEqual(t, updatedAccount.Name, *params.Name)
	})

	t.Run("empty params still bumps updated_at", func(t *testing.T) {
		account, err := db.CreateAccount("Account1", 100.0)
		testutil.AssertNoError(t, err)

		updatedAccount, err := db.UpdateAccount(account.Id, storage.UpdateAccountParams{})
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, updatedAccount.ManualAdjustment, 0.0)
		testutil.AssertEqual(t, account.Name, updatedAccount.Name)
	})

	t.Run("wrong account id return not found", func(t *testing.T) {
		_, err := db.UpdateAccount(uuid.NewString(), storage.UpdateAccountParams{})
		testutil.AssertErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestDeleteAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("existing account", func(t *testing.T) {
		acc, err := db.CreateAccount("Account1", 100.0)
		testutil.AssertNoError(t, err)

		err = db.DeleteAccount(acc.Id)
		testutil.AssertNoError(t, err)
	})

	t.Run("non existing account", func(t *testing.T) {
		err := db.DeleteAccount(uuid.NewString())
		testutil.AssertErrorIs(t, err, storage.ErrAccountNotFound)
	})

	t.Run("double delete account", func(t *testing.T) {
		acc, err := db.CreateAccount("Account1", 100.0)
		testutil.AssertNoError(t, err)
		err = db.DeleteAccount(acc.Id)
		testutil.AssertNoError(t, err)
		err = db.DeleteAccount(acc.Id)
		testutil.AssertErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestGetAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	testAccount, err := db.CreateAccount("Account1", 100.0)
	testutil.AssertNoError(t, err)

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
				testutil.AssertErrorIs(t, err, tc.expectedErr)
				return
			}

			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, account.Id, tc.id)
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
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(accounts), 0)
	})

	t.Run("existing accounts in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		createdAccounts, err := createTestAccounts(db)
		testutil.AssertNoError(t, err)
		accounts, err := db.GetAccounts()
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(accounts), len(createdAccounts))
	})
}
