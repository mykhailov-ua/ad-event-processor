package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"log"
	"net/http"
	"os"

	"espx/internal/licensing/vendorserver"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	port := flag.String("port", "8120", "Port to run vendor license server on")
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "Database DSN")
	flag.Parse()

	if *dsn == "" {
		*dsn = "postgres://postgres:postgres@localhost:5432/espx?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer pool.Close()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate signing key: %v", err)
	}

	srv := vendorserver.New(pool, priv)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Printf("Vendor License Server running on port %s", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
