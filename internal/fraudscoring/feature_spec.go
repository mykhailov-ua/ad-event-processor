package fraudscoring

var FeatureNames = []string{
	"events",
	"clicks",
	"ctr",
	"spend_norm",
	"spend_ratio",
	"unique_users",
	"unique_uas",
}

func Dims() int {
	return len(FeatureNames)
}
