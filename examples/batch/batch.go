package main

import (
	"context"
	"errors"
	"fmt"

	pgx "github.com/jackc/pgx/v5"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
)

type PgxIface interface {
	Begin(context.Context) (pgx.Tx, error)
	Close()
}

func databaseSetup(db PgxIface) (err error) {

	// Initialize a database object
	tx, err := db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("databaseSetup: %s", err)
	}
	// Finally, commit changes or rollback
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(context.Background())
		default:
			_ = tx.Rollback(context.Background())
		}
	}()

	// Create a new table 'ledger'
	sql := `CREATE TABLE IF NOT EXISTS ledger (
		id SERIAL PRIMARY KEY,
		description VARCHAR(255) NOT NULL,
		amount INTEGER NOT NULL);`

	// Execute SQL commands
	_, err = tx.Exec(context.Background(), sql)
	if err != nil {
		return fmt.Errorf("databaseSetup: %s", err)
	}

	return
}

func requestBatch(db PgxIface) (err error) {

	// Initialize a database object
	tx, err := db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("requestBatch: %s", err)
	}
	// Finally, commit changes or rollback
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(context.Background())
		default:
			_ = tx.Rollback(context.Background())
		}
	}()

	// Create a Batch object
	batch := &pgx.Batch{}

	// Add SQL commands to queue
	batch.Queue("insert into ledger(description, amount) values($1, $2)", "q1", 1)
	batch.Queue("insert into ledger(description, amount) values($1, $2)", "q2", 2)
	batch.Queue("insert into ledger(description, amount) values($1, $2)", "q3", 3)
	batch.Queue("select id, description, amount from ledger order by id")
	batch.Queue("select id, description, amount from ledger order by amount")
	batch.Queue("select * from ledger where false")
	batch.Queue("select sum(amount) from ledger")

	// Create a BatchRequest object
	br := tx.SendBatch(context.Background(), batch)
	if br == nil {
		return errors.New("SendBatch returns a NIL object")
	}
	defer br.Close()

	// Execute a BatchRequest
	_, err = br.Exec()
	if err != nil {
		return fmt.Errorf("requestBatch: %s", err)
	}

	return
}

func databaseCleanup(db PgxIface) (err error) {

	// Initialize a database object
	tx, err := db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("databaseCleanup: %s", err)
	}
	// Finally, commit changes or rollback
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(context.Background())
		default:
			_ = tx.Rollback(context.Background())
		}
	}()

	// Delete all rows in a table
	sql := `DELETE FROM ledger ;`

	// Execute SQL commands
	_, err = tx.Exec(context.Background(), sql)
	if err != nil {
		return fmt.Errorf("databaseCleanup: %s", err)
	}

	return
}

func main() {

	// @NOTE: the real connection is not required for tests
	db, err := pgxpool.New(context.Background(), "postgres://<rolename>:<password>@<hostname>/<database>")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create a database table
	if err = databaseSetup(db); err != nil {
		panic(err)
	}

	// Create and send a batch request
	if err = requestBatch(db); err != nil {
		panic(err)
	}

	// Delete all rows in table
	if err = databaseCleanup(db); err != nil {
		panic(err)
	}
}
