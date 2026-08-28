package campaign

const defaultCloneNameSuffix = " (copy)"

func CloneCampaignName(sourceName, prefix, suffix string) string {
	if prefix != "" {
		return prefix + sourceName
	}
	if suffix != "" {
		return sourceName + suffix
	}
	return sourceName + defaultCloneNameSuffix
}

func cloneCampaignName(sourceName, prefix, suffix string) string {
	return CloneCampaignName(sourceName, prefix, suffix)
}

func NormalizedCloneOptions(opts CloneCampaignOptions) CloneCampaignOptions {
	if !opts.IncludeFlow && !opts.IncludePostbacks && !opts.IncludeFraud && !opts.IncludePlacementBlocks && !opts.ResetSpend {
		return CloneCampaignOptions{
			IncludeFlow:            true,
			IncludePostbacks:       true,
			IncludeFraud:           true,
			IncludePlacementBlocks: true,
		}
	}
	return opts
}

func normalizedCloneOptions(opts CloneCampaignOptions) CloneCampaignOptions {
	return NormalizedCloneOptions(opts)
}
