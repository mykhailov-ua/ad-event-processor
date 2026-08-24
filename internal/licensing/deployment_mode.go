package licensing

import "fmt"

const (
	DeploymentModeOnPrem      = "on_prem"
	DeploymentModeManagedSaas = "managed_saas"
)

func NormalizeDeploymentMode(mode string) string {
	switch mode {
	case DeploymentModeManagedSaas:
		return DeploymentModeManagedSaas
	default:
		return DeploymentModeOnPrem
	}
}

func ValidateDeploymentMode(mode string) error {
	switch mode {
	case "", DeploymentModeOnPrem, DeploymentModeManagedSaas:
		return nil
	default:
		return fmt.Errorf("invalid deployment_mode %q", mode)
	}
}
