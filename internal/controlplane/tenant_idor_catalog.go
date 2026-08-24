package controlplane

import "fmt"

func tenantIsolationProbePaths(victimCustomerID, campaignID string) []string {
	from := "2026-01-01"
	to := "2026-12-31"
	return []string{
		"/api/v1/customers/" + victimCustomerID + "/balance",
		"/api/v1/customers/" + victimCustomerID,
		"/api/v1/customers/" + victimCustomerID + "/ledger",
		"/api/v1/customers/" + victimCustomerID + "/balance/export?format=csv",
		fmt.Sprintf("/api/v1/billing/usage/export?format=csv&from=%s&to=%s&customer_id=%s", from, to, victimCustomerID),
		"/api/v1/team/overview?customer_id=" + victimCustomerID,
		"/api/v1/campaigns/" + campaignID + "/stats",
	}
}
