package netintel

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"

	"github.com/oschwald/maxminddb-golang"
)

var (
	ErrInvalidIP         = errors.New("invalid IP")
	ErrGeoProviderClosed = errors.New("geoip provider closed")
)

type GeoProvider interface {
	GetCountry(ip string) (string, error)
	IsAnonymous(ip string) (bool, error)
	Close() error
}

type countryResult struct {
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

type anonymousIPResult struct {
	IsAnonymous       bool `maxminddb:"is_anonymous"`
	IsAnonymousVPN    bool `maxminddb:"is_anonymous_vpn"`
	IsHostingProvider bool `maxminddb:"is_hosting_provider"`
	IsPublicProxy     bool `maxminddb:"is_public_proxy"`
	IsTorExitNode     bool `maxminddb:"is_tor_exit_node"`
}

type asnResult struct {
	ASN uint `maxminddb:"autonomous_system_number"`
}

var countryPool = sync.Pool{
	New: func() any {
		return &countryResult{}
	},
}

var anonymousIPPool = sync.Pool{
	New: func() any {
		return &anonymousIPResult{}
	},
}

var asnPool = sync.Pool{
	New: func() any {
		return &asnResult{}
	},
}

// MaxMindProvider: mmap MaxMind DB readers; GeoIPWatcher hot-swaps under RWMutex without restart.
type MaxMindProvider struct {
	reader    *maxminddb.Reader
	asnReader *maxminddb.Reader
	mu        sync.RWMutex
}

func NewMaxMindProvider(dbPath string) (*MaxMindProvider, error) {
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open maxmind db: %w", err)
	}
	return &MaxMindProvider{reader: db}, nil
}

func (p *MaxMindProvider) GetCountry(ipStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", ErrInvalidIP
	}

	p.mu.RLock()
	reader := p.reader
	p.mu.RUnlock()
	if reader == nil {
		return "", ErrGeoProviderClosed
	}

	record := countryPool.Get().(*countryResult)
	record.Country.IsoCode = ""
	defer countryPool.Put(record)

	if err := reader.Lookup(ip, record); err != nil {
		return "", err
	}

	return record.Country.IsoCode, nil
}

func (p *MaxMindProvider) IsAnonymous(ipStr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, ErrInvalidIP
	}

	p.mu.RLock()
	reader := p.reader
	p.mu.RUnlock()
	if reader == nil {
		return false, ErrGeoProviderClosed
	}

	record := anonymousIPPool.Get().(*anonymousIPResult)
	record.IsAnonymous = false
	record.IsAnonymousVPN = false
	record.IsHostingProvider = false
	record.IsPublicProxy = false
	record.IsTorExitNode = false
	defer anonymousIPPool.Put(record)

	if err := reader.Lookup(ip, record); err != nil {
		return false, err
	}

	return record.IsAnonymous || record.IsAnonymousVPN || record.IsHostingProvider || record.IsPublicProxy || record.IsTorExitNode, nil
}

func (p *MaxMindProvider) LookupASN(ipStr string) (uint32, bool) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, false
	}

	p.mu.RLock()
	reader := p.asnReader
	p.mu.RUnlock()
	if reader == nil {
		return 0, false
	}

	record := asnPool.Get().(*asnResult)
	record.ASN = 0
	defer asnPool.Put(record)

	if err := reader.Lookup(ip, record); err != nil || record.ASN == 0 {
		return 0, false
	}
	return uint32(record.ASN), true
}

func (p *MaxMindProvider) ReloadASN(dbPath string) error {
	if p == nil {
		return fmt.Errorf("geoip provider is nil")
	}
	if dbPath == "" {
		return nil
	}
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open asn db: %w", err)
	}

	p.mu.Lock()
	old := p.asnReader
	p.asnReader = db
	p.mu.Unlock()

	if old != nil {
		return old.Close()
	}
	return nil
}

func (p *MaxMindProvider) Reload(dbPath string) error {
	if p == nil {
		return fmt.Errorf("geoip provider is nil")
	}
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open maxmind db: %w", err)
	}

	p.mu.Lock()
	old := p.reader
	p.reader = db
	p.mu.Unlock()

	if old != nil {
		return old.Close()
	}
	return nil
}

func (p *MaxMindProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var firstErr error
	if p.asnReader != nil {
		if err := p.asnReader.Close(); err != nil {
			firstErr = err
		}
		p.asnReader = nil
	}
	if p.reader != nil {
		err := p.reader.Close()
		p.reader = nil
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type MockGeoProvider struct {
	Countries map[string]string
	ASN       map[string]uint32
}

func (p *MockGeoProvider) GetCountry(ip string) (string, error) {
	if p != nil {
		if code, ok := p.Countries[ip]; ok {
			return code, nil
		}
	}
	return "US", nil
}

func (p *MockGeoProvider) IsAnonymous(ip string) (bool, error) {
	return strings.HasSuffix(ip, ".66") || strings.HasSuffix(ip, ".77"), nil
}

func (p *MockGeoProvider) LookupASN(ip string) (uint32, bool) {
	if p == nil || p.ASN == nil {
		return 0, false
	}
	asn, ok := p.ASN[ip]
	return asn, ok
}

func (p *MockGeoProvider) Close() error { return nil }

var countryPrimaryTimezone = map[string]string{
	"US": "America/New_York",
	"GB": "Europe/London",
	"DE": "Europe/Berlin",
	"FR": "Europe/Paris",
	"UA": "Europe/Kyiv",
	"RU": "Europe/Moscow",
	"IN": "Asia/Kolkata",
	"JP": "Asia/Tokyo",
	"AU": "Australia/Sydney",
	"BR": "America/Sao_Paulo",
	"CA": "America/Toronto",
	"NL": "Europe/Amsterdam",
	"PL": "Europe/Warsaw",
	"IT": "Europe/Rome",
	"ES": "Europe/Madrid",
}

func timezoneOffsetHours(tz string, ts time.Time) (int, bool) {
	if tz == "" {
		return 0, false
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return 0, false
	}
	_, offset := ts.In(loc).Zone()
	return offset / 3600, true
}

func TimezoneMismatchHours(browserTZ, country string, now time.Time) (mismatch bool, deltaHours int) {
	expected, ok := countryPrimaryTimezone[strings.ToUpper(country)]
	if !ok || browserTZ == "" {
		return false, 0
	}
	expOff, okExp := timezoneOffsetHours(expected, now)
	gotOff, okGot := timezoneOffsetHours(browserTZ, now)
	if !okExp || !okGot {
		return false, 0
	}
	delta := expOff - gotOff
	if delta < 0 {
		delta = -delta
	}
	return delta > 2, delta
}

type GeoIPUpdaterConfig struct {
	DBPath         string
	StagingPath    string
	EditionID      string
	LicenseKey     string
	UpdateInterval time.Duration
	HTTPClient     *http.Client
}

type GeoIPUpdater struct {
	cfg GeoIPUpdaterConfig
}

func NewGeoIPUpdater(cfg GeoIPUpdaterConfig) *GeoIPUpdater {
	if cfg.DBPath == "" {
		cfg.DBPath = "deploy/geoip/GeoLite2-Country.mmdb"
	}
	if cfg.StagingPath == "" {
		cfg.StagingPath = cfg.DBPath + ".staging"
	}
	if cfg.EditionID == "" {
		cfg.EditionID = "GeoLite2-Country"
	}
	if cfg.UpdateInterval <= 0 {
		cfg.UpdateInterval = 24 * time.Hour
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Minute}
	}
	return &GeoIPUpdater{cfg: cfg}
}

func (u *GeoIPUpdater) Start(ctx context.Context) {
	if u == nil {
		return
	}

	ticker := time.NewTicker(u.cfg.UpdateInterval)
	defer ticker.Stop()

	u.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.runOnce(ctx)
		}
	}
}

func (u *GeoIPUpdater) runOnce(ctx context.Context) {
	if u.cfg.LicenseKey == "" {
		slog.Debug("geoip updater skipped: MAXMIND_LICENSE_KEY not configured")
		return
	}

	if err := u.downloadAndInstall(ctx); err != nil {
		metrics.GeoIPUpdateErrorsTotal.Inc()
		slog.Warn("geoip updater cycle failed", "error", err)
		return
	}
	slog.Info("geoip database refreshed", "path", u.cfg.DBPath)
}

func (u *GeoIPUpdater) downloadAndInstall(ctx context.Context) error {
	url := fmt.Sprintf(
		"https://download.maxmind.com/app/geoip_download?edition_id=%s&license_key=%s&suffix=tar.gz",
		u.cfg.EditionID,
		u.cfg.LicenseKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := u.cfg.HTTPClient.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return fmt.Errorf("download maxmind archive: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("maxmind download status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := os.MkdirAll(filepath.Dir(u.cfg.StagingPath), 0o755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(u.cfg.StagingPath), "geoip-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp archive: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("write archive: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	archive, err := os.Open(tmpName)
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()

	gzr, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	var extracted bool
	for {
		hdr, readErr := tr.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read tar: %w", readErr)
		}
		if hdr.Typeflag != tar.TypeReg || !strings.HasSuffix(hdr.Name, ".mmdb") {
			continue
		}

		out, createErr := os.OpenFile(u.cfg.StagingPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if createErr != nil {
			return fmt.Errorf("create staging mmdb: %w", createErr)
		}
		if _, copyErr := io.Copy(out, tr); copyErr != nil {
			_ = out.Close()
			return fmt.Errorf("extract mmdb: %w", copyErr)
		}
		if closeErr := out.Close(); closeErr != nil {
			return closeErr
		}
		extracted = true
		break
	}
	if !extracted {
		return fmt.Errorf("no .mmdb file found in maxmind archive")
	}

	if err := os.Rename(u.cfg.StagingPath, u.cfg.DBPath); err != nil {
		return fmt.Errorf("atomic install: %w", err)
	}
	return nil
}

type GeoIPWatcher struct {
	provider    *MaxMindProvider
	countryPath string
	asnPath     string
	interval    time.Duration
}

func NewGeoIPWatcher(provider *MaxMindProvider, countryDBPath, asnDBPath string, interval time.Duration) *GeoIPWatcher {
	if interval <= 0 {
		interval = time.Minute
	}
	return &GeoIPWatcher{
		provider:    provider,
		countryPath: countryDBPath,
		asnPath:     asnDBPath,
		interval:    interval,
	}
}

func (w *GeoIPWatcher) Start(ctx context.Context) {
	if w == nil || w.provider == nil {
		return
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	var lastCountryMod, lastASNMod time.Time
	if info, err := os.Stat(w.countryPath); err == nil {
		lastCountryMod = info.ModTime()
	}
	if w.asnPath != "" {
		if info, err := os.Stat(w.asnPath); err == nil {
			lastASNMod = info.ModTime()
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.countryPath != "" {
				info, err := os.Stat(w.countryPath)
				if err != nil {
					slog.Debug("geoip watcher stat failed", "path", w.countryPath, "error", err)
				} else if info.ModTime().After(lastCountryMod) {
					if err := w.provider.Reload(w.countryPath); err != nil {
						metrics.GeoIPReloadErrorsTotal.Inc()
						slog.Warn("geoip hot reload failed", "path", w.countryPath, "error", err)
					} else {
						lastCountryMod = info.ModTime()
						slog.Info("geoip database hot-reloaded", "path", w.countryPath, "mtime", lastCountryMod.UTC().Format(time.RFC3339))
					}
				}
			}
			if w.asnPath != "" {
				info, err := os.Stat(w.asnPath)
				if err != nil {
					slog.Debug("geoip asn watcher stat failed", "path", w.asnPath, "error", err)
				} else if info.ModTime().After(lastASNMod) {
					if err := w.provider.ReloadASN(w.asnPath); err != nil {
						metrics.GeoIPReloadErrorsTotal.Inc()
						slog.Warn("geoip asn hot reload failed", "path", w.asnPath, "error", err)
					} else {
						lastASNMod = info.ModTime()
						slog.Info("geoip asn database hot-reloaded", "path", w.asnPath, "mtime", lastASNMod.UTC().Format(time.RFC3339))
					}
				}
			}
		}
	}
}

var countryPrimaryLang = map[string][2]byte{
	"AU": {'e', 'n'},
	"BR": {'p', 't'},
	"CA": {'e', 'n'},
	"DE": {'d', 'e'},
	"ES": {'e', 's'},
	"FR": {'f', 'r'},
	"GB": {'e', 'n'},
	"IN": {'h', 'i'},
	"IT": {'i', 't'},
	"JP": {'j', 'a'},
	"NL": {'n', 'l'},
	"PL": {'p', 'l'},
	"PT": {'p', 't'},
	"RU": {'r', 'u'},
	"UA": {'u', 'k'},
	"US": {'e', 'n'},
}

type acceptLangTag struct {
	base   [2]byte
	region [2]byte
}

func parseAcceptLanguageTags(acceptLang string, out []acceptLangTag) int {
	if acceptLang == "" {
		return 0
	}
	count := 0
	start := 0
	n := len(acceptLang)
	for i := 0; i <= n; i++ {
		if i < n && acceptLang[i] != ',' {
			continue
		}
		token := trimAcceptLangToken(acceptLang[start:i])
		if len(token) > 0 && count < len(out) {
			if tag, ok := parseAcceptLangTag(token); ok {
				out[count] = tag
				count++
			}
		}
		start = i + 1
	}
	return count
}

func trimAcceptLangToken(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	if i := strings.IndexByte(s, ';'); i >= 0 {
		s = s[:i]
		for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
			s = s[:len(s)-1]
		}
	}
	return s
}

func parseAcceptLangTag(token string) (acceptLangTag, bool) {
	var tag acceptLangTag
	if len(token) < 2 {
		return tag, false
	}
	if token[0] >= '0' && token[0] <= '9' {
		return tag, false
	}
	if len(token) > 2 && token[2] == '-' {
		if len(token) < 5 {
			return tag, false
		}
		tag.base[0] = foldASCIILower(token[0])
		tag.base[1] = foldASCIILower(token[1])
		tag.region[0] = foldASCIIUpper(token[3])
		tag.region[1] = foldASCIIUpper(token[4])
		return tag, true
	}
	if len(token) != 2 {
		return tag, false
	}
	tag.base[0] = foldASCIILower(token[0])
	tag.base[1] = foldASCIILower(token[1])
	return tag, true
}

func foldASCIILower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func foldASCIIUpper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}

func AcceptLangGeoMismatch(acceptLang, geoCountry string) bool {
	if acceptLang == "" || len(geoCountry) != 2 {
		return false
	}
	expected, ok := countryPrimaryLang[geoCountry]
	if !ok {
		return false
	}
	var geo [2]byte
	geo[0] = foldASCIIUpper(geoCountry[0])
	geo[1] = foldASCIIUpper(geoCountry[1])

	var tags [8]acceptLangTag
	n := parseAcceptLanguageTags(acceptLang, tags[:])
	if n == 0 {
		return false
	}
	if tags[0].region[0] != 0 && tags[0].region == geo {
		return false
	}
	for i := range n {
		if tags[i].base == expected {
			return false
		}
	}
	return true
}

const (
	ConnTimingRTTBit  uint8 = 1 << 0
	ConnTimingTTFBBit uint8 = 1 << 1
)

func GeoHashFromCountry(country string) uint32 {
	if len(country) < 2 {
		return 0
	}
	c0, c1 := country[0]|0x20, country[1]|0x20
	return uint32(c0)<<8 | uint32(c1)
}

func EnsureIngestGeo(geo GeoProvider, evt *domain.Event) {
	if geo == nil || evt == nil || evt.IP == "" || evt.IngestGeoResolved {
		return
	}
	evt.IngestGeoResolved = true
	country, err := geo.GetCountry(evt.IP)
	if err == nil && country != "" {
		evt.GeoCountry = country
		evt.GeoHash = GeoHashFromCountry(country)
	}
	if anon, anonErr := geo.IsAnonymous(evt.IP); anonErr == nil {
		evt.IngestAnonymous = anon
	}
}
