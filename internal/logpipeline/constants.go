package logpipeline

const (
	readySuffix      = ".log.zst.ready"
	evacuatingSuffix = ".log.zst.evacuating"
	compactingSuffix = ".compacting"
	warmTmpSuffix    = ".tmp"
	filteredTmpExt   = ".filtered.tmp"
)
