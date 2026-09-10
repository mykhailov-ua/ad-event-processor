// Package mask enforces server-side mutation deny rules for masked campaign roles.
//
// Role:
// - Rejects PATCH/PUT changing protected economics and URL fields when authz.MaskLevel is masked.
//
// Verify:
// go test ./internal/campaign/mask/ -short -run TestMaskedMutation -count=1
package mask
