package assuredworkloads

const (
	BTPRegionDammamGCP = "cf-sa30"
)

func IsKSA(platformRegion string) bool {
	if platformRegion == BTPRegionDammamGCP {
		return true
	}
	return false
}
