package installer

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/bidshard/ad-event-processor/pkg/branding"
	"github.com/bidshard/ad-event-processor/pkg/runtimepaths"
)

const trackerUnitTemplate = `[Unit]
Description={{.ProductName}} Tracker
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=10
StartLimitBurst=3
OnFailure=ad-event-processor-rollback@tracker.service

[Service]
Type=simple
EnvironmentFile=-{{.SecretsEnv}}
Environment=TRACKER_INGRESS_SCHEMA={{.IngressSchema}}
Environment=GOGC={{.GOGC}}
Environment=GOMEMLIMIT={{.GOMEMLIMIT}}
ExecStart=/usr/local/bin/tracker
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`

func renderSystemdUnit(profile *InstallProfile) ([]byte, error) {
	tmpl, err := template.New("tracker").Parse(trackerUnitTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	data := map[string]string{
		"ProductName":   branding.ProductName(),
		"Interface":     profile.Interface,
		"Profile":       string(profile.Type),
		"IngressSchema": trackerIngressSchema(profile),
		"GOGC":          trackerGOGC,
		"GOMEMLIMIT":    trackerGOMEMLIMIT,
		"SecretsEnv":    runtimepaths.SecretsEnvPath(),
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	if profile.EdgeXDP {
		buf.WriteString(fmt.Sprintf("\n# edge_xdp enabled on %s\n", profile.Interface))
	}
	return buf.Bytes(), nil
}
