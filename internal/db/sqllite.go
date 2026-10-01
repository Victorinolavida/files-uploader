package db

import (
	"database/sql"
	"log"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"

	_ "modernc.org/sqlite"
)

func NewDB() *bun.DB {
	sqlDB, err := sql.Open("sqlite", "./local.db")
	if err != nil {
		log.Fatal(err)
	}

	db := bun.NewDB(
		sqlDB,
		sqlitedialect.New(),
	).WithQueryHook(
		bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		),
	)

	return db
}
