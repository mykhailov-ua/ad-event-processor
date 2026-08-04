package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type doctorAPIResponse struct {
	Overall          string `json:"overall"`
	Checks           []doctorAPICheck
	ClickURLTemplate string `json:"click_url_template"`
	TrackingDomain   string `json:"tracking_domain"`
}

type doctorAPICheck struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms"`
}

var doctorRemediation = map[string]string{
	"redis":          "docker compose logs redis-0 redis-1 redis-2 redis-3",
	"postgres":       "docker compose logs db",
	"clickhouse":     "docker compose logs clickhouse",
	"dns":            "point tracking domain A-record to this host; ingress/TLS is not bundled in single_vps",
	"tracker":        "docker compose logs tracker-0",
	"control":        "docker compose logs control",
	"processor":      "docker compose logs processor",
	"redis_password": "set REDIS_PASSWORD in .env (not the example placeholder), then bash scripts/install/bidshard-install.sh apply",
	"pii_salt":       "set PII_SALT_HEX in .env to a unique 64-char hex value",
}

func RunDoctor(asJSON bool) error {
	root := repoRoot()
	loadDotEnv(root)

	depsOK := true
	depsDetail := ""
	script := checkDepsScript()
	cmd := exec.Command("bash", script)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "ROOT="+root)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		depsOK = false
		depsDetail = strings.TrimSpace(stderr.String())
		if depsDetail == "" {
			depsDetail = strings.TrimSpace(stdout.String())
		}
	}

	apiResp, apiErr := fetchDoctorAPI()
	results := map[string]string{
		"dependencies": "OK",
		"topology":     "OK",
	}
	if !depsOK {
		results["dependencies"] = "FAIL"
		results["topology"] = depsDetail
		if hint := remediationForDetail(depsDetail); hint != "" {
			results["topology"] = depsDetail + " | fix: " + hint
		}
	}
	if apiErr != nil {
		results["api_doctor"] = "FAIL: " + apiErr.Error()
	} else if apiResp != nil {
		results["api_doctor"] = apiResp.Overall
		if apiResp.Overall != "pass" {
			for _, c := range apiResp.Checks {
				if c.Status == "fail" || c.Status == "warn" {
					line := fmt.Sprintf("%s: %s (%s)", c.ID, c.Status, c.Message)
					if hint := doctorRemediation[c.ID]; hint != "" {
						line += " | fix: " + hint
					}
					results["check_"+c.ID] = line
				}
			}
		}
	}

	if asJSON {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		if !depsOK {
			return fmt.Errorf("doctor: dependency checks failed")
		}
		if apiResp != nil && apiResp.Overall == "fail" {
			return fmt.Errorf("doctor: platform checks failed")
		}
		return nil
	}

	fmt.Println("Running doctor health checks...")
	for k, v := range results {
		fmt.Printf("%s: %s\n", k, v)
	}
	if !depsOK {
		fmt.Println("hint: bash scripts/dev/stack.sh status")
		return fmt.Errorf("doctor: dependency checks failed")
	}
	if apiResp != nil && apiResp.Overall == "fail" {
		fmt.Println("hint: curl -H \"X-Admin-API-Key: $ADMIN_API_KEY\" http://127.0.0.1:" + managementPort() + "/api/v1/ops/doctor")
		return fmt.Errorf("doctor: platform checks failed")
	}
	return nil
}

func fetchDoctorAPI() (*doctorAPIResponse, error) {
	key := strings.TrimSpace(os.Getenv("ADMIN_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("ADMIN_API_KEY not set")
	}
	url := managementBaseURL() + "/api/v1/ops/doctor"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(headerAdminAPIKey, key)
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET doctor %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out doctorAPIResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func remediationForDetail(detail string) string {
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "postgresql"):
		return doctorRemediation["postgres"]
	case strings.Contains(lower, "clickhouse"):
		return doctorRemediation["clickhouse"]
	case strings.Contains(lower, "redis"):
		return doctorRemediation["redis"]
	default:
		return "bash scripts/dev/stack.sh single-vps"
	}
}
