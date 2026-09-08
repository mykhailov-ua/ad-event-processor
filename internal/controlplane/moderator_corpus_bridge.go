package controlplane

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/moderatorcorpus"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (s *Service) ModeratorCorpusPool() *pgxpool.Pool { return s.GetPool() }

func (s *Service) ModeratorCorpusClickHouse() *database.ClickHouseQuery { return s.clickhouseQuery }

func (s *Service) ModeratorCorpusFeedDir() string {
	if s == nil || s.cfg == nil {
		return "/var/lib/ad-event-processor/moderator-corpus"
	}
	dir := s.cfg.ModeratorCorpusFeedDir
	if dir == "" {
		dir = "/var/lib/ad-event-processor/moderator-corpus"
	}
	return dir
}

func (s *Service) ModeratorCorpusFeedLastRefresh(ctx context.Context) (string, bool) {
	path := filepath.Join(s.ModeratorCorpusFeedDir(), "moderator_corpus.txt")
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	return info.ModTime().UTC().Format(time.RFC3339), true
}

func (s *Service) RefreshModeratorCorpusFeed(ctx context.Context) error {
	if s == nil || s.GetPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	rows, err := s.GetPool().Query(ctx, `
		SELECT ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count
		FROM fraud_moderator_corpus
		ORDER BY updated_at DESC`)
	if err != nil {
		return fmt.Errorf("load fraud_moderator_corpus: %w", err)
	}
	defer rows.Close()
	entries := make([]moderatorcorpus.Entry, 0, 64)
	for rows.Next() {
		var ja3, ja4, tcpSig, webgl string
		var desync int16
		if err := rows.Scan(&ja3, &ja4, &tcpSig, &webgl, &desync); err != nil {
			return err
		}
		entry, err := moderatorcorpus.EntryFromTuple(moderatorcorpus.Tuple{
			JA3:              ja3,
			JA4:              ja4,
			TCPSig:           tcpSig,
			WebGLRenderer:    webgl,
			LayerDesyncCount: uint8(desync),
		})
		if err != nil {
			return err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	dir := s.ModeratorCorpusFeedDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir moderator corpus feed dir: %w", err)
	}
	path := filepath.Join(dir, "moderator_corpus.txt")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, moderatorcorpus.FormatFeed(entries), 0o644); err != nil {
		return fmt.Errorf("write moderator corpus feed: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("publish moderator corpus feed: %w", err)
	}
	return nil
}
