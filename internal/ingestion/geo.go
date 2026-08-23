package ingestion

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

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
