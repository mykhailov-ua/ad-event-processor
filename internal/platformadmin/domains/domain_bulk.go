package domains

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/pkg/platformconfig"

	"github.com/google/uuid"
)

const (
	domainBulkMaxHostnames = 500
	domainBulkMinInterval  = 6 * time.Second
)

type DomainBulkRequest struct {
	Hostnames        []string   `json:"hostnames"`
	CSV              string     `json:"csv,omitempty"`
	CloudflareZoneID string     `json:"cloudflare_zone_id"`
	PoolID           *uuid.UUID `json:"pool_id,omitempty"`
}

type DomainBulkJobRow struct {
	Hostname string `json:"hostname"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
}

type DomainBulkJobStatus struct {
	JobID     string             `json:"job_id"`
	Kind      string             `json:"kind"`
	Status    string             `json:"status"`
	Total     int                `json:"total"`
	Completed int                `json:"completed"`
	Failed    int                `json:"failed"`
	Results   []DomainBulkJobRow `json:"results,omitempty"`
	Error     string             `json:"error,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type domainBulkJobRecord struct {
	status DomainBulkJobStatus
	spec   domainBulkJobSpec
}

type domainBulkJobSpec struct {
	kind             string
	hostnames        []string
	cloudflareZoneID string
	poolID           *uuid.UUID
}

func normalizeBulkHostnames(raw []string, csvText string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	appendHost := func(line string) {
		host := platformconfig.ResolveHost(line)
		if host == "" {
			return
		}
		if _, ok := seen[host]; ok {
			return
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	for _, h := range raw {
		appendHost(h)
	}
	csvText = strings.TrimSpace(csvText)
	if csvText != "" {
		for _, line := range strings.Split(csvText, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if comma := strings.Index(line, ","); comma >= 0 {
				line = strings.TrimSpace(line[:comma])
			}
			if strings.EqualFold(line, "hostname") {
				continue
			}
			appendHost(line)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one hostname is required")
	}
	if len(out) > domainBulkMaxHostnames {
		return nil, fmt.Errorf("too many hostnames (max %d)", domainBulkMaxHostnames)
	}
	return out, nil
}

func (dh *DomainHealth) bulkJobsMap() map[string]*domainBulkJobRecord {
	if dh == nil {
		return nil
	}
	dh.bulkMu.Lock()
	defer dh.bulkMu.Unlock()
	if dh.bulkJobs == nil {
		dh.bulkJobs = make(map[string]*domainBulkJobRecord)
	}
	return dh.bulkJobs
}

func (dh *DomainHealth) StartBulkParkProbe(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	hosts, err := normalizeBulkHostnames(req.Hostnames, req.CSV)
	if err != nil {
		return DomainBulkJobStatus{}, err
	}
	zoneID := strings.TrimSpace(req.CloudflareZoneID)
	if zoneID == "" {
		return DomainBulkJobStatus{}, fmt.Errorf("cloudflare_zone_id is required")
	}
	return dh.startBulkJob(ctx, domainBulkJobSpec{
		kind:             "park_probe",
		hostnames:        hosts,
		cloudflareZoneID: zoneID,
		poolID:           req.PoolID,
	})
}

func (dh *DomainHealth) StartBulkSSL(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	hosts, err := normalizeBulkHostnames(req.Hostnames, req.CSV)
	if err != nil {
		return DomainBulkJobStatus{}, err
	}
	return dh.startBulkJob(ctx, domainBulkJobSpec{
		kind:      "ssl",
		hostnames: hosts,
	})
}

func (dh *DomainHealth) startBulkJob(ctx context.Context, spec domainBulkJobSpec) (DomainBulkJobStatus, error) {
	if dh == nil || dh.host == nil {
		return DomainBulkJobStatus{}, fmt.Errorf("service unavailable")
	}
	jobID := uuid.NewString()
	now := time.Now().UTC()
	rec := &domainBulkJobRecord{
		spec: spec,
		status: DomainBulkJobStatus{
			JobID:     jobID,
			Kind:      spec.kind,
			Status:    "pending",
			Total:     len(spec.hostnames),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	jobs := dh.bulkJobsMap()
	if jobs == nil {
		return DomainBulkJobStatus{}, fmt.Errorf("service unavailable")
	}
	jobs[jobID] = rec

	dh.host.StartBackgroundWorker(func() { //nolint:contextcheck // bulk domain job outlives HTTP request
		dh.runBulkJob(context.Background(), jobID)
	})
	return rec.status, nil
}

func (dh *DomainHealth) GetBulkJob(_ context.Context, jobID string) (DomainBulkJobStatus, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return DomainBulkJobStatus{}, fmt.Errorf("job_id is required")
	}
	jobs := dh.bulkJobsMap()
	if jobs == nil {
		return DomainBulkJobStatus{}, fmt.Errorf("service unavailable")
	}
	dh.bulkMu.RLock()
	rec, ok := jobs[jobID]
	dh.bulkMu.RUnlock()
	if !ok {
		return DomainBulkJobStatus{}, fmt.Errorf("job not found")
	}
	return rec.status, nil
}

func (dh *DomainHealth) runBulkJob(ctx context.Context, jobID string) {
	dh.bulkMu.Lock()
	rec, ok := dh.bulkJobs[jobID]
	if !ok {
		dh.bulkMu.Unlock()
		return
	}
	rec.status.Status = "running"
	rec.status.UpdatedAt = time.Now().UTC()
	spec := rec.spec
	dh.bulkMu.Unlock()

	for i, host := range spec.hostnames {
		row := DomainBulkJobRow{Hostname: host}
		switch spec.kind {
		case "park_probe":
			_, err := dh.ParkDomain(ctx, ParkDomainRequest{
				Domain:           host,
				CloudflareZoneID: spec.cloudflareZoneID,
				PoolID:           spec.poolID,
			})
			if err != nil {
				row.Error = err.Error()
			} else {
				_, probeErr := dh.ProbeDomainNow(ctx, host)
				if probeErr != nil {
					row.Error = probeErr.Error()
				} else {
					row.OK = true
				}
			}
		case "ssl":
			result, err := dh.SetupDomainSSL(ctx, host)
			//nolint:gocritic // ifElseChain: ssl setup status mapping
			if err != nil {
				row.Error = err.Error()
			} else if result.Status == "failed" {
				row.Error = result.Message
				if row.Error == "" {
					row.Error = "ssl setup failed"
				}
			} else {
				row.OK = true
			}
		default:
			row.Error = "unknown bulk job kind"
		}

		dh.bulkMu.Lock()
		if rec, ok = dh.bulkJobs[jobID]; ok {
			rec.status.Completed++
			if !row.OK {
				rec.status.Failed++
			}
			rec.status.Results = append(rec.status.Results, row)
			rec.status.UpdatedAt = time.Now().UTC()
		}
		dh.bulkMu.Unlock()

		if i < len(spec.hostnames)-1 {
			time.Sleep(domainBulkMinInterval)
		}
	}

	dh.bulkMu.Lock()
	if rec, ok = dh.bulkJobs[jobID]; ok {
		rec.status.Status = "completed"
		if rec.status.Failed == rec.status.Total {
			rec.status.Status = "failed"
			rec.status.Error = "all hostnames failed"
		}
		rec.status.UpdatedAt = time.Now().UTC()
	}
	dh.bulkMu.Unlock()
}
