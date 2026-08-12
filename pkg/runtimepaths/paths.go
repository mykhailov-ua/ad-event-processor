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
