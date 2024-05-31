package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	pgx "github.com/jackc/pgx/v5"
	pgconn "github.com/jackc/pgx/v5/pgconn"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
)

type PgxIface interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Close()
}

func databaseSetup(db PgxIface) (err error) {
	// Create a new table 'ledger'
	sql := `CREATE TABLE IF NOT EXISTS ledger (
		id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
		description TEXT NOT NULL,
		amount BIGINT NOT NULL);`
	_, err = db.Exec(context.Background(), sql)
	return err
}

func iterateResults(br pgx.BatchResults, QueuedQueries []*pgx.QueuedQuery) error {

	file, err := os.Create("OUTPUT.md")
	if err != nil {
		return fmt.Errorf("iterateResults: %s", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "## PostgreSQL Batch Example output\n")

	// Iterate over a batch of queued queries
	for _, query := range QueuedQueries {

		// Print SQL field of the current query
		fmt.Fprintf(file, "### %v \n", query.SQL)

		//  reads results from the current query
		rows, err := br.Query()
		if err != nil {
			return fmt.Errorf("iterateResults: %s", err)
		}

		// Print column headers
		fmt.Fprintf(file, "- *DESCRIPTION* , *AMOUNT* \n")
		// Iterate over the resulted rows
		//
		var id, amount, descr = int64(0), int64(0), string("")
		_, err = pgx.ForEachRow(rows, []any{&id, &descr, &amount}, func() error {
			fmt.Fprintf(file, "- \"%v\" , %d \n", descr, amount)
			return nil
		})
		if err != nil {
			return fmt.Errorf("iterateResults: %s", err)
		}
	}

	return err
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
	batch.Queue(
		`INSERT INTO ledger(description, amount) VALUES ($1, $2), ($3, $4)`,
		"first item", 1, "second item", 2)

	batch.Queue("SELECT * FROM ledger")
	batch.Queue("SELECT * FROM ledger WHERE amount = 1")

	// Efficiently transmits queued queries as a single transaction.
	// After the queries are run, a BatchResults object is returned.
	//
	br := tx.SendBatch(context.Background(), batch)
	if br == nil {
		return errors.New("SendBatch returns a NIL object")
	}
	defer br.Close()

	// Read the first query
	_, err = br.Exec()
	if err != nil {
		return err
	}
	// Iterate over batch results and queries.
	// Note: the first query is left out of the queue.
	//
	return iterateResults(br, batch.QueuedQueries[1:])

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

	// Delete all rows in table ledger
	sql := `DELETE FROM ledger ;`

	// Execute SQL commands
	_, err = tx.Exec(context.Background(), sql)

	return err
}

func main() {

	// @NOTE: the real connection is not required for tests
	db, err := pgxpool.New(context.Background(), "postgres://<user>:<password>@<hostname>/<database>")
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

	// Delete all rows in table ledger
	if err = databaseCleanup(db); err != nil {
		panic(err)
	}
}
