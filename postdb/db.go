package postdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
)

// Dial creates a client connection to the database.
func Dial(
	host, name, login, password string,
	handleFatalError func(error)) (DB, error) {
	conn, err := pgx.Connect(context.Background(),
		fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable",
			login, password, host, name))
	if err != nil {
		return nil, err
	}
	return &pgDB{handleFatalError: handleFatalError, conn: conn}, nil
}

// Message represents message record in a database.
type Message struct {
	ID   uint32
	Kind uint32
	Size uint32
}

// Post represents post record in a database.
type Post struct {
	ID                 uint32
	Author             uint32
	Time               time.Time
	LocaltionLongitude float64
	LocaltionLatitude  float64
	Messages           []*Message
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
	InsertPost(id, author uint32, latitude, longitude float64) error
	// InsertMessage inserts new message record into database.
	InsertMessage(id, post, kind, size uint32) error
	// AppendMessage appends message data.
	AppendMessage(id, post uint32, data []byte) error
}

// DB represents a database connection.
type DB interface {
	// Close closes database connection and frees resources.
	Close()
	// Begin starts a new database transaction to execute database IO operations.
	Begin() (DBTrans, error)
	// GetPost returns post by ID.
	GetPost(id uint32) (*Post, error)
}

////////////////////////////////////////////////////////////////////////////////

type dbTrans struct {
	db *pgDB
	tx pgx.Tx
}

func (trans *dbTrans) Commit() error {
	if trans.tx == nil {
		return errors.New("failed to commit as tx is closed")
	}
	if err := trans.tx.Commit(context.Background()); err != nil {
		return err
	}
	trans.tx = nil
	return nil
}

func (trans *dbTrans) Rollback() {
	if trans.tx == nil {
		// Already committed. Tx allows to call Rollback after Commit, but it
		// returns error "tx is closed" in this case and we can't see was it real
		// error or not.
		return
	}
	if err := trans.tx.Rollback(context.Background()); err != nil {
		// There is no way to restore application state at error at rollback, the
		// behavior is undefined, so the application must be stopped.
		trans.db.handleFatalError(
			fmt.Errorf(`failed to rollback database transaction: "%s"`,
				err))
	}
}

func (trans *dbTrans) InsertPost(
	id, author uint32,
	latitude, longitude float64) error {
	_, err := trans.tx.Exec(context.Background(),
		"INSERT INTO post(id, author, location) VALUES($1, $2, point($3, $4))",
		id, author, latitude, longitude)
	return err
}

func (trans *dbTrans) InsertMessage(id, post, kind, size uint32) error {
	_, err := trans.tx.Exec(context.Background(),
		"INSERT INTO message(id, post, kind, size) VALUES($1, $2, $3, $4)",
		id, post, kind, size)
	return err
}

func (trans *dbTrans) AppendMessage(id, post uint32, data []byte) error {
	_, err := trans.tx.Exec(context.Background(),
		"UPDATE message SET data = data || $3 WHERE id = $1 AND post = $2",
		id, post, data)
	return err
}

////////////////////////////////////////////////////////////////////////////////

type pgDB struct {
	handleFatalError func(error) // todo: replace it by au10.Log when we will not have recursive  dependencies (by au10.Query, for ex)
	conn             *pgx.Conn
}

func (db *pgDB) Close() {
	db.conn.Close(context.Background())
}

func (db *pgDB) Begin() (DBTrans, error) {
	tx, err := db.conn.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	return &dbTrans{db: db, tx: tx}, nil
}

func (db *pgDB) GetPost(id uint32) (*Post, error) {
	rows, err := db.conn.Query(context.Background(),
		"SELECT post.author, message.id, message.kind, message.size FROM post LEFT JOIN message ON post.id = message.post WHERE post.id = $1",
		id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*Message{}
	var author *uint32
	for rows.Next() {
		var messageID *uint32
		var messageKind *uint32
		var messageSize *uint32
		if author == nil {
			err = rows.Scan(&author, &messageID, &messageKind, &messageSize)
		} else {
			err = rows.Scan(nil, &messageID, &messageKind, &messageSize)
		}
		if err != nil {
			return nil, err
		}
		if messageID != nil {
			messages = append(messages, &Message{
				ID:   *messageID,
				Kind: *messageKind,
				Size: *messageSize})
		}
	}

	if author == nil {
		return nil, fmt.Errorf("post with ID %d does not exists", id)
	}

	return &Post{
			ID:       id,
			Author:   *author,
			Messages: messages},
		nil
}

////////////////////////////////////////////////////////////////////////////////
