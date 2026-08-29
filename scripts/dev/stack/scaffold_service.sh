#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <service-name>"
  exit 1
fi

SERVICE_NAME=$1
ROOT_DIR=$(git rev-parse --show-toplevel)
INTERNAL_DIR="$ROOT_DIR/internal/$SERVICE_NAME"
CMD_DIR="$ROOT_DIR/cmd/$SERVICE_NAME"
PKG_NAME=$(echo "$SERVICE_NAME" | tr '-' '_')

echo "Scaffolding service: $SERVICE_NAME"

mkdir -p "$INTERNAL_DIR"/{db,pb,queries,migrations}
mkdir -p "$CMD_DIR"

SCHEMA_NAME="$(echo "$PKG_NAME" | tr '-' '_')"

cat > "$INTERNAL_DIR/migrations/00001_init.sql" << EOF
-- +goose Up
CREATE SCHEMA IF NOT EXISTS ${SCHEMA_NAME};

-- +goose Down
DROP SCHEMA IF EXISTS ${SCHEMA_NAME} CASCADE;
EOF

cat << EOF > "$INTERNAL_DIR/service.go"
package $PKG_NAME

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"ad-event-processor/internal/$SERVICE_NAME/db"
)

type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (s *Service) Start(ctx context.Context) error {
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	return nil
}
EOF

cat << EOF > "$INTERNAL_DIR/handler.go"
package $PKG_NAME

import (
	"net/http"
)

func RegisterHandlers(mux *http.ServeMux, s *Service) {
	mux.HandleFunc("GET /api/v1/$SERVICE_NAME/health", s.handleHealth)
}

func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
EOF

cat << EOF > "$INTERNAL_DIR/queries/models.sql"
-- Empty models file for sqlc
EOF

cat << EOF > "$CMD_DIR/main.go"
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"$PKG_NAME" "ad-event-processor/internal/$SERVICE_NAME"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres:5430/ad_event_processor?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	service := $PKG_NAME.NewService(pool)

	mux := http.NewServeMux()
	$PKG_NAME.RegisterHandlers(mux, service)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := service.Start(ctx); err != nil {
			log.Printf("service start error: %v", err)
		}
	}()

	go func() {
		log.Printf("starting $SERVICE_NAME service on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down $SERVICE_NAME service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	if err := service.Stop(shutdownCtx); err != nil {
		log.Printf("service stop error: %v", err)
	}
}
EOF

cat << EOF >> "$ROOT_DIR/sqlc.yaml"
  - schema: "internal/$SERVICE_NAME/migrations"
    queries: "internal/$SERVICE_NAME/queries"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "internal/$SERVICE_NAME/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
EOF

echo "Service $SERVICE_NAME scaffolded successfully"
echo "Run 'task gen' to generate sqlc code"
