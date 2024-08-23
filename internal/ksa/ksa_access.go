package ksa

const (
	BTPRegionDammamGCP = "cf-sa30"
)

func IsKSARestrictedAccess(platformRegion string) bool {
	if platformRegion == BTPRegionDammamGCP {
		return true
	}
	return false
}
