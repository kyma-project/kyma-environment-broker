package euaccess

const (
	BTPRegionSwitzerlandAzure           = "cf-ch20"
	BTPRegionEuropeAWS                  = "cf-eu11"
	BTPRegionFrankfurtSapConvergedCloud = "cf-eu01"
	BTPRegionRotSapConvergedCloud       = "cf-eu02"
)

func IsEURestrictedAccess(platformRegion string) bool {
	switch platformRegion {
	case BTPRegionSwitzerlandAzure, BTPRegionEuropeAWS, BTPRegionFrankfurtSapConvergedCloud, BTPRegionRotSapConvergedCloud:
		return true
	default:
		return false
	}
}
