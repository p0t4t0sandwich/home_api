package database

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// -------------- Globals --------------
var DATABASE_URL = os.Getenv("DATABASE_URL")

// -------------- Functions --------------

// GetDB - Get a connection pool to the database
func GetDB(database string) *pgxpool.Pool {
	if DATABASE_URL == "" {
		log.Fatal("DATABASE_URL is not set")
		return nil
	}

	PgPool, err := pgxpool.New(context.Background(), DATABASE_URL+"/"+database)
	if err != nil {
		log.Fatal("Unable to create connection pool:", err)
		return nil
	}
	return PgPool
}
