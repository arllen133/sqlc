package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/04_transactions/models"
	_ "github.com/arllen133/sqlc/examples/04_transactions/models/generated"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dsn := "file:test_tx.db?cache=shared&mode=rwc"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS accounts (id INTEGER PRIMARY KEY, balance INTEGER);`); err != nil {
		log.Fatal(err)
	}

	sess := sqlc.NewSession(db, &sqlc.SQLiteDialect{})
	ctx := context.Background()

	repo := sqlc.NewRepository[models.Account](sess)

	// Setup accounts
	acct1 := &models.Account{Balance: 1000}
	acct2 := &models.Account{Balance: 500}
	repo.Create(ctx, acct1)
	repo.Create(ctx, acct2)

	fmt.Printf("Before: Acct1=%d, Acct2=%d\n", acct1.Balance, acct2.Balance)

	// Transaction: Transfer 100 from Acct1 to Acct2
	fmt.Println("--- Transferring 100 ---")
	err = sess.Transaction(ctx, func(txSess *sqlc.Session) error {
		txRepo := sqlc.NewRepository[models.Account](txSess)

		// 1. Deduct from Acct1
		// Refresh
		a1, err := txRepo.FindOne(ctx, acct1.ID)
		if err != nil {
			return err
		}

		if a1.Balance < 100 {
			return fmt.Errorf("insufficient funds")
		}
		a1.Balance -= 100
		if err := txRepo.Update(ctx, a1); err != nil {
			return err
		}

		// 2. Add to Acct2
		a2, err := txRepo.FindOne(ctx, acct2.ID)
		if err != nil {
			return err
		}

		a2.Balance += 100
		if err := txRepo.Update(ctx, a2); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Fatal("Transaction failed:", err)
	}

	// Verify
	repo = sqlc.NewRepository[models.Account](sess) // Reset to normal session (though original repo is also fine if session didn't close)
	a1Final, _ := repo.FindOne(ctx, acct1.ID)
	a2Final, _ := repo.FindOne(ctx, acct2.ID)
	fmt.Printf("After:  Acct1=%d, Acct2=%d\n", a1Final.Balance, a2Final.Balance)

	os.Remove("test_tx.db")
}
