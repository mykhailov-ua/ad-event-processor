package commandpalette

func campaignStatusLabel(status string) string {
	switch status {
	case "ACTIVE":
		return "Active"
	case "PAUSED":
		return "Paused"
	case "ARCHIVED":
		return "Archived"
	default:
		if status == "" {
			return "Unknown"
		}
		return status
	}
}

func campaignStatusTone(status string) string {
	switch status {
	case "ACTIVE":
		return "success"
	case "PAUSED":
		return "warning"
	case "ARCHIVED":
		return "muted"
	default:
		return "muted"
	}
}
