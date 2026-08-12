package installer

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/bidshard/ad-event-processor/pkg/branding"
	"github.com/bidshard/ad-event-processor/pkg/runtimepaths"
)

const edgeXDPUnitTemplate = `[Unit]
Description={{.ProductName}} edge XDP attach
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=INGRESS_INTERFACE={{.Interface}}
Environment=BPF_PIN_DIR={{.BPFPinDir}}
ExecStart=/usr/local/bin/edge-xdp
AmbientCapabilities=CAP_BPF CAP_NET_ADMIN
CapabilityBoundingSet=CAP_BPF CAP_NET_ADMIN
NoNewPrivileges=true
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`

const edgeBPFSyncUnitTemplate = `[Unit]
Description={{.ProductName}} edge BPF blacklist sync
After=network-online.target {{.EdgeXDPUnit}}
Requires={{.EdgeXDPUnit}}
Wants=network-online.target

[Service]
Type=simple
Environment=BPF_PIN_DIR={{.BPFPinDir}}
EnvironmentFile=-{{.SecretsEnv}}
ExecStart=/usr/local/bin/edge-bpf-sync
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`

type edgeUnitData struct {
	ProductName string
	Interface   string
	BPFPinDir   string
	SecretsEnv  string
	EdgeXDPUnit string
}

func renderEdgeXDPUnit(profile *InstallProfile) ([]byte, error) {
	tmpl, err := template.New("edge-xdp").Parse(edgeXDPUnitTemplate)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	data := edgeUnitData{
		ProductName: branding.ProductName(),
		Interface:   profile.Interface,
		BPFPinDir:   edgeBPFPinDir,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderEdgeBPFSyncUnit(profile *InstallProfile) ([]byte, error) {
	tmpl, err := template.New("edge-bpf-sync").Parse(edgeBPFSyncUnitTemplate)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	data := edgeUnitData{
		ProductName: branding.ProductName(),
		SecretsEnv:  runtimepaths.SecretsEnvPath(),
		BPFPinDir:   edgeBPFPinDir,
		EdgeXDPUnit: EdgeXDPSystemdUnitName,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type renderedManifest struct {
	path    string
	content []byte
	mode    os.FileMode
}

func edgeSystemdManifests(profile *InstallProfile) ([]renderedManifest, error) {
	if profile == nil || !profile.EdgeXDP || profile.Type != ProfileSingleVPS {
		return nil, nil
	}
	xdp, err := renderEdgeXDPUnit(profile)
	if err != nil {
		return nil, err
	}
	sync, err := renderEdgeBPFSyncUnit(profile)
	if err != nil {
		return nil, err
	}
	return []renderedManifest{
		{systemdUnitPath(EdgeXDPSystemdUnitName), xdp, 0o644},
		{systemdUnitPath(EdgeBPFSyncSystemdUnitName), sync, 0o644},
	}, nil
}

func syncEdgeSystemdUnits(profile *InstallProfile, dryRun bool) error {
	if profile.Type != ProfileSingleVPS {
		return nil
	}
	if dryRun {
		if profile.EdgeXDP {
			fmt.Printf("[Dry-Run] Would systemctl daemon-reload && enable --now %s %s\n",
				EdgeXDPSystemdUnitName, EdgeBPFSyncSystemdUnitName)
		} else {
			fmt.Printf("[Dry-Run] Would stop and disable %s %s\n",
				EdgeBPFSyncSystemdUnitName, EdgeXDPSystemdUnitName)
		}
		return nil
	}
	if profile.EdgeXDP {
		return enableEdgeSystemdUnits()
	}
	return disableEdgeSystemdUnits()
}

func enableEdgeSystemdUnits() error {
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	for _, unit := range []string{EdgeXDPSystemdUnitName, EdgeBPFSyncSystemdUnitName} {
		cmd := exec.Command("systemctl", "enable", "--now", unit)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl enable --now %s: %w (%s)", unit, err, out)
		}
		fmt.Printf("Enabled %s\n", unit)
	}
	return nil
}

// disableEdgeSystemdUnits stops edge XDP services; Nginx Lua blacklist remains active.
func disableEdgeSystemdUnits() error {
	for _, unit := range []string{EdgeBPFSyncSystemdUnitName, EdgeXDPSystemdUnitName} {
		_ = exec.Command("systemctl", "stop", unit).Run()
		_ = exec.Command("systemctl", "disable", unit).Run()
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
