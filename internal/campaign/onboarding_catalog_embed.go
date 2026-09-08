package campaign

import _ "embed"

// Default onboarding catalog baked into control binaries when deploy/schemas is absent
// (distroless images). Canonical editable copy: deploy/schemas/onboarding/catalog.v1.yaml.
//
//go:embed onboarding/catalog.v1.yaml
var embeddedOnboardingCatalogYAML []byte
