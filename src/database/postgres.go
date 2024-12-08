package database

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// -------------- Globals --------------
var POSTGRES_URI = os.Getenv("POSTGRES_URI")

// -------------- Functions --------------

// GetDB - Get a connection pool to the database
func GetDB(database string) *pgxpool.Pool {
	if POSTGRES_URI == "" {
		log.Fatal("POSTGRES_URI is not set")
		return nil
	}

	PgPool, err := pgxpool.New(context.Background(), POSTGRES_URI+"/"+database)
	if err != nil {
		log.Fatal("Unable to create connection pool:", err)
		return nil
	}
	return PgPool
}
