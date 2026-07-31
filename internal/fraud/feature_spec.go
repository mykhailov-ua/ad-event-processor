package fraud

var FeatureNames = []string{
	"events",
	"clicks",
	"ctr",
	"spend_norm",
	"spend_ratio",
	"unique_users",
	"unique_uas",
	"events_per_ua",
	"clicks_per_ua",
	"users_per_ua",
	"clicks_per_user",
	"spend_per_click",
	"ua_diversity",
	"events_per_user",
	"impression_pressure",
	"user_click_gap",
}

func Dims() int {
	return len(FeatureNames)
}
