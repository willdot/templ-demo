package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/glebarez/go-sqlite"
)

type database struct {
	db *sql.DB
}

func newDatabase(dbPath string) (*database, error) {
	db, err := createDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create db: %w", err)
	}
	return &database{db: db}, nil
}

func createDbFile(dbFilename string) error {
	if _, err := os.Stat(dbFilename); !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	f, err := os.Create(dbFilename)
	if err != nil {
		return fmt.Errorf("create db file : %w", err)
	}
	f.Close()
	return nil
}

func createDatabase(dbPath string) (*sql.DB, error) {
	err := createDbFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create db file: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	err = createUsersTable(db)
	if err != nil {
		return nil, fmt.Errorf("create users table: %w", err)
	}

	return db, nil
}

func createUsersTable(db *sql.DB) error {
	createUsersTableSQL := `CREATE TABLE IF NOT EXISTS users (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"username" TEXT,
		"password" TEXT,
		UNIQUE(username)
	  );`

	slog.Info("Create users table...")
	statement, err := db.Prepare(createUsersTableSQL)
	if err != nil {
		return fmt.Errorf("prepare DB statement to create users table: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("exec sql statement to create users table: %w", err)
	}
	slog.Info("users table created")

	return nil
}

var errUserNotFound = errors.New("user not found")
var errIncorrectPassword = errors.New("password incorrect")

func (d *database) loginUser(username, password string) error {
	sqlQ := "SELECT password FROM users WHERE username = ?"
	row := d.db.QueryRow(sqlQ, username, password)
	resultPassword := ""

	err := row.Scan(&resultPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return errUserNotFound
		}
		return fmt.Errorf("reading row: %w", err)
	}

	if resultPassword != password {
		return errIncorrectPassword
	}

	return nil
}

func (d *database) createUser(username, password string) error {
	// Yh I'm storing passwords in plain text AND WHAT? THIS IS A FUCKING DEMO APP
	sqlQ := `INSERT INTO users (username, password) VALUES (?, ?);`
	_, err := d.db.Exec(sqlQ, username, password)
	if err != nil {
		return fmt.Errorf("exec insert user: %w", err)
	}
	return nil
}
