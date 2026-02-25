package assuredworkloads

const (
	BTPRegionDammamGCP = "cf-sa30"
)

func IsKSA(platformRegion string) bool {
	return platformRegion == BTPRegionDammamGCP
}
