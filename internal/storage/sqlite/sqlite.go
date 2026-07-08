package sqlite

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	db.SetMaxOpenConns(1)

	return &Storage{db: db}, nil
}

func (s *Storage) RunMigrations() error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	defer src.Close()

	drv, err := sqlite3.WithInstance(s.db, &sqlite3.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", drv)
	if err != nil {
		return err
	}

	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		err = nil
	}
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}
