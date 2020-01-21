package postdb

import (
	"context"
	"fmt"
	"log"

	"bitbucket.org/au10/service/au10"
	"github.com/jackc/pgx/v4"
)

// Dial creates a client connection to the database.
func Dial(host, name, login, password string) (DB, error) {
	conn, err := pgx.Connect(context.Background(),
		fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable",
			login, password, host, name))
	if err != nil {
		return nil, err
	}
	return &pgDB{conn: conn}, nil
}

// DBTrans represents an active interface to a database in one transaction. All
// actions are collected by transaction and has to be written by commit
// operations.
type DBTrans interface {
	// Commit commits all changes since the start if the rollback was not called
	// before.
	Commit() error
	// Rollback rollbacks all changes since the start if the commit was not called
	// before.
	Rollback()
	// InsertPost inserts new post record into database.
	InsertPost(au10.Vocal) error
}

// DB represents a database connection.
type DB interface {
	// Close closes database connection and frees resources.
	Close()

	// Begin starts a new database transaction to execute database IO operations.
	Begin() (DBTrans, error)
}

////////////////////////////////////////////////////////////////////////////////

type dbTrans struct {
	tx pgx.Tx
}

func (trans *dbTrans) Commit() error {
	return trans.tx.Commit(context.Background())
}

func (trans *dbTrans) Rollback() {
	if err := trans.tx.Rollback(context.Background()); err != nil {
		// There is no way to restore application state at error at rollback, the
		// behavior is undefined, so the application must be stopped.
		log.Panicf(`Failed to commit database transaction: "%s".`, err)
	}
}

func (trans *dbTrans) InsertPost(post au10.Vocal) error {
	location := post.GetLocation()
	_, err := trans.tx.Exec(context.Background(),
		"INSERT INTO post(id, author, location) VALUES($1, $2, point($3, $4))",
		post.GetID(), post.GetAuthor(), location.Latitude, location.Longitude)
	return err
}

////////////////////////////////////////////////////////////////////////////////

type pgDB struct {
	conn *pgx.Conn
}

func (db *pgDB) Close() {
	db.conn.Close(context.Background())
}

func (db *pgDB) Begin() (DBTrans, error) {
	tx, err := db.conn.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	return &dbTrans{tx: tx}, nil
}

////////////////////////////////////////////////////////////////////////////////
