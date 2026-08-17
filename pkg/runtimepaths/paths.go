// Package runtimepaths resolves standard runtime directory locations.
package runtimepaths

import "fmt"

const (
	legacyStackSlug     = "es" + "px"
	EtcDir              = "/etc/ad-event-processor"
	RunDir              = "/run/ad-event-processor"
	ComposeRunVolume    = "ad_event_processor_run"
	ContainerLicense    = EtcDir + "/license.jwt"
	LegacyEtcDir        = "/etc/" + legacyStackSlug
	LegacyRunDir        = "/run/" + legacyStackSlug
	LegacyComposeVolume = legacyStackSlug + "_run"
)

func SecretsEnvPath() string { return EtcDir + "/secrets.env" }

func LicensePath() string { return EtcDir + "/license.jwt" }

func PostgresSocketDir() string { return RunDir + "/postgresql" }

func RedisSocket(shard int) string {
	return fmt.Sprintf("%s/redis/redis-%d.sock", RunDir, shard)
}

func BrokerGnetSocket() string {
	return RunDir + "/broker/gnet.sock"
}

func BrokerHealthSocket() string {
	return RunDir + "/broker/health.sock"
}

func RegionProxyGnetSocket() string {
	return RunDir + "/region-proxy/gnet.sock"
}

func RegionProxyHealthSocket() string {
	return RunDir + "/region-proxy/health.sock"
}

func ClickHouseNativeSocket() string {
	return RunDir + "/clickhouse/native.sock"
}

func ControlHTTPSocket() string {
	return RunDir + "/control/http.sock"
}

func TrackerSocket(instance int) string {
	return fmt.Sprintf("%s/tracker/tracker-%d.sock", RunDir, instance)
}
