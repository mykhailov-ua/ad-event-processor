package domains

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
)

const (
	acmeLEProductionDirectory = "https://acme-v02.api.letsencrypt.org/directory"
	acmeLEStagingDirectory    = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

type wildcardIssueRequest struct {
	ZoneName    string
	IncludeApex bool
	Email       string
	ZoneID      string
	Directory   string
}

type wildcardIssueResult struct {
	CertPEM  []byte
	KeyPEM   []byte
	NotAfter time.Time
}

type wildcardCertIssuer interface {
	Issue(ctx context.Context, cf DomainCloudflareClient, req wildcardIssueRequest) (wildcardIssueResult, error)
}

type acmeDNS01Issuer struct{}

func (a *acmeDNS01Issuer) Issue(ctx context.Context, cf DomainCloudflareClient, req wildcardIssueRequest) (wildcardIssueResult, error) {
	if cf == nil {
		return wildcardIssueResult{}, errors.New("cloudflare client unavailable")
	}
	zoneName := normalizeZoneName(req.ZoneName)
	if zoneName == "" {
		return wildcardIssueResult{}, errors.New("zone_name is required")
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return wildcardIssueResult{}, errors.New("acme email is required")
	}
	directory := strings.TrimSpace(req.Directory)
	if directory == "" {
		directory = acmeLEProductionDirectory
	}

	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return wildcardIssueResult{}, fmt.Errorf("acme account key: %w", err)
	}
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return wildcardIssueResult{}, fmt.Errorf("acme cert key: %w", err)
	}

	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: directory,
	}
	account := &acme.Account{Contact: []string{"mailto:" + email}}
	account, err = client.Register(ctx, account, acme.AcceptTOS)
	if err != nil {
		return wildcardIssueResult{}, fmt.Errorf("acme register: %w", err)
	}
	client.KID = acme.KeyID(account.URI)

	identifiers := []acme.AuthzID{{Type: "dns", Value: "*." + zoneName}}
	if req.IncludeApex {
		identifiers = append(identifiers, acme.AuthzID{Type: "dns", Value: zoneName})
	}
	order, err := client.AuthorizeOrder(ctx, identifiers)
	if err != nil {
		return wildcardIssueResult{}, fmt.Errorf("acme authorize order: %w", err)
	}

	cleanupFns := make([]func() error, 0, len(order.AuthzURLs))
	defer func() {
		for _, fn := range cleanupFns {
			if fn != nil {
				_ = fn()
			}
		}
	}()

	for _, authURL := range order.AuthzURLs {
		authz, err := client.GetAuthorization(ctx, authURL)
		if err != nil {
			return wildcardIssueResult{}, fmt.Errorf("acme get authorization: %w", err)
		}
		challenge, err := pickDNS01Challenge(authz)
		if err != nil {
			return wildcardIssueResult{}, err
		}
		txtValue, err := client.DNS01ChallengeRecord(challenge.Token)
		if err != nil {
			return wildcardIssueResult{}, fmt.Errorf("acme dns01 record: %w", err)
		}
		fqdn := acmeChallengeFQDN(authz.Identifier.Value)
		recordID, err := cf.UpsertTXTRecord(ctx, req.ZoneID, fqdn, txtValue)
		if err != nil {
			return wildcardIssueResult{}, fmt.Errorf("cloudflare acme txt: %w", err)
		}
		cleanupFns = append(cleanupFns, func() error {
			return cf.DeleteDNSRecord(context.Background(), req.ZoneID, recordID)
		})
		if _, err := client.Accept(ctx, challenge); err != nil {
			return wildcardIssueResult{}, fmt.Errorf("acme accept challenge: %w", err)
		}
		if _, err := client.WaitAuthorization(ctx, authz.URI); err != nil {
			return wildcardIssueResult{}, fmt.Errorf("acme wait authorization: %w", err)
		}
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "*." + zoneName},
		DNSNames: wildcardCSRNames(zoneName, req.IncludeApex),
	}, certKey)
	if err != nil {
		return wildcardIssueResult{}, fmt.Errorf("acme csr: %w", err)
	}
	derChains, _, err := client.CreateOrderCert(ctx, order.FinalizeURL, csr, true)
	if err != nil {
		return wildcardIssueResult{}, fmt.Errorf("acme finalize order: %w", err)
	}
	if len(derChains) == 0 || len(derChains[0]) == 0 {
		return wildcardIssueResult{}, errors.New("acme finalize order: empty certificate")
	}
	der := derChains[0]
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return wildcardIssueResult{}, fmt.Errorf("acme parse cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM, err := encodePrivateKeyPEM(certKey)
	if err != nil {
		return wildcardIssueResult{}, err
	}
	return wildcardIssueResult{
		CertPEM:  certPEM,
		KeyPEM:   keyPEM,
		NotAfter: cert.NotAfter.UTC(),
	}, nil
}

func pickDNS01Challenge(authz *acme.Authorization) (*acme.Challenge, error) {
	for _, ch := range authz.Challenges {
		if ch.Type == "dns-01" && ch.Status != acme.StatusInvalid {
			return ch, nil
		}
	}
	return nil, fmt.Errorf("acme dns-01 challenge missing for %q", authz.Identifier.Value)
}

func acmeChallengeFQDN(identifier string) string {
	zone := strings.TrimPrefix(strings.TrimSpace(identifier), "*.")
	return "_acme-challenge." + zone
}

func wildcardCSRNames(zoneName string, includeApex bool) []string {
	names := []string{"*." + zoneName}
	if includeApex {
		names = append(names, zoneName)
	}
	return names
}

func encodePrivateKeyPEM(key crypto.Signer) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func normalizeZoneName(zone string) string {
	zone = strings.TrimSpace(strings.ToLower(zone))
	zone = strings.TrimPrefix(zone, "*.")
	return strings.TrimSuffix(zone, ".")
}

func hostnameMatchesWildcardZone(host, zoneName string, includeApex bool) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	zoneName = normalizeZoneName(zoneName)
	if host == "" || zoneName == "" {
		return false
	}
	if includeApex && host == zoneName {
		return true
	}
	return strings.HasSuffix(host, "."+zoneName)
}
