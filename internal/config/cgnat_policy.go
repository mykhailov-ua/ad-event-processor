package config

func (c *Config) CGNATMobileIPBypassEnabled() bool {
	return c != nil && c.CGNATMobileIPBypass
}
