package domain

import "testing"

func TestParseRedirectComplianceMode(t *testing.T) {
	if ParseRedirectComplianceMode("legacy_dmr") != RedirectComplianceLegacyDMR {
		t.Fatal("legacy_dmr")
	}
	if ParseRedirectComplianceMode("strict") != RedirectComplianceStrict {
		t.Fatal("strict")
	}
	if ParseRedirectComplianceMode("") != RedirectComplianceStrict {
		t.Fatal("empty defaults strict")
	}
	camp := &Campaign{
		DmrEnabled:             true,
		RedirectComplianceMode: RedirectComplianceStrict,
	}
	if camp.RedirectAllowsDMR() {
		t.Fatal("strict must not allow DMR")
	}
	camp.RedirectComplianceMode = RedirectComplianceLegacyDMR
	if !camp.RedirectAllowsDMR() {
		t.Fatal("legacy_dmr must allow DMR")
	}
}
