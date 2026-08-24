package installer

import (
	"errors"
	"fmt"
	"os"

	"ad-event-processor/pkg/naming"
)

type Profile string

const (
	ProfileSingleVPS  Profile = "single_vps"
	ProfileComposeDev Profile = "compose_dev"
)

type IngressSchema string

const (
	IngressSchemaOpenRTB3               IngressSchema = "openrtb_3"
	IngressSchemaAdEventProcessorNative IngressSchema = "ad_event_processor_native"
)

func legacyIngressNativeSchema() IngressSchema {
	return IngressSchema(naming.DeprecatedIngressNativeSchema())
}

type ServiceDeploy struct {
	Binary    string `yaml:"binary,omitempty"`
	HealthURL string `yaml:"health_url,omitempty"`
	Version   string `yaml:"version,omitempty"`
}

type InstallProfile struct {
	Type             Profile       `yaml:"profile"`
	IngressSchema    IngressSchema `yaml:"ingress_schema"`
	EdgeXDP          bool          `yaml:"edge_xdp"`
	MultiRegion      bool          `yaml:"multi_region"`
	TelemetryEnabled bool          `yaml:"telemetry_enabled"`
	Interface        string        `yaml:"interface"`
	Tracker          ServiceDeploy `yaml:"tracker,omitempty"`
	Processor        ServiceDeploy `yaml:"processor,omitempty"`
}

func (p *InstallProfile) Validate() error {
	switch p.Type {
	case ProfileSingleVPS, ProfileComposeDev:
	default:
		return fmt.Errorf("invalid profile: %s", p.Type)
	}

	if p.EdgeXDP && p.Type == ProfileComposeDev {
		return errors.New("edge_xdp is not supported in compose_dev profile")
	}

	if p.EdgeXDP && !btfAvailable() {
		return errors.New("edge_xdp requires BTF support (PF-BTF)")
	}

	if p.MultiRegion && p.Type == ProfileComposeDev {
		return errors.New("multi_region is not supported in compose_dev profile")
	}

	if p.Interface == "" && p.Type == ProfileSingleVPS {
		return errors.New("network interface must be specified for production-like profiles")
	}

	if p.IngressSchema == "" {
		p.IngressSchema = IngressSchemaOpenRTB3
	}
	switch p.IngressSchema {
	case IngressSchemaOpenRTB3, IngressSchemaAdEventProcessorNative, legacyIngressNativeSchema(), "native_v1":
	default:
		return fmt.Errorf("invalid ingress_schema: %s (want openrtb_3 or ad_event_processor_native)", p.IngressSchema)
	}
	if p.IngressSchema == legacyIngressNativeSchema() || p.IngressSchema == "native_v1" {
		p.IngressSchema = IngressSchemaAdEventProcessorNative
	}

	return nil
}

func btfAvailable() bool {
	_, err := os.Stat(btfSysPath())
	return err == nil
}
