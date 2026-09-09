package reportjob

const (
	ExportRowLimitMin           = 1
	ExportRowLimitDefault       = 100_000
	ExportRowLimitMax           = 5_000_000
	ExportRowLimitLicenseGated  = 1000
	ExportRowLimitChunkBytesLow = 2 * 1024 * 1024
)

func ExportRowLimitMaxForChunkBytes(maxExportChunkBytes uint64) int {
	if maxExportChunkBytes > 0 && maxExportChunkBytes < ExportRowLimitChunkBytesLow {
		return ExportRowLimitLicenseGated
	}
	return ExportRowLimitMax
}

func ResolveExportRowLimitTierMax(chunkBytes uint64, licenseGatedReport bool) int {
	tierMax := ExportRowLimitMaxForChunkBytes(chunkBytes)
	if licenseGatedReport && tierMax > ExportRowLimitLicenseGated {
		return ExportRowLimitLicenseGated
	}
	return tierMax
}

func NormalizeExportRowLimit(requested int, tierMax int) int {
	if tierMax <= 0 {
		tierMax = ExportRowLimitMax
	}
	if requested <= 0 {
		if ExportRowLimitDefault < tierMax {
			return ExportRowLimitDefault
		}
		return tierMax
	}
	if requested < ExportRowLimitMin {
		return ExportRowLimitMin
	}
	if requested > tierMax {
		return tierMax
	}
	return requested
}
