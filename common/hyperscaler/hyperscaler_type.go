package hyperscaler

const (
	BTPRegionDammamGCP = "cf-sa30"
)

type Type struct {
	hyperscalerName   string
	hyperscalerRegion string
	platformRegion    string
}

func GCP(platformRegion string) Type {
	return Type{
		hyperscalerName: "gcp",
		platformRegion:  platformRegion,
	}
}

func Azure() Type {
	return Type{
		hyperscalerName: "azure",
	}
}

func AWS() Type {
	return Type{
		hyperscalerName: "aws",
	}
}

func SapConvergedCloud(region string) Type {
	return Type{
		hyperscalerName:   "openstack",
		hyperscalerRegion: region,
	}
}

func (t Type) GetName() string {
	return t.hyperscalerName
}

// TODO remove when regions are mandatory, and the hack in resolve_creds is no longer needed
func (t Type) GetRegion() string {
	return t.hyperscalerRegion
}

func (t Type) GetKey() string {
	if t.hyperscalerName == "openstack" && t.hyperscalerRegion != "" {
		return t.hyperscalerName + "_" + t.hyperscalerRegion
	}
	if t.hyperscalerName == "gcp" && t.platformRegion == BTPRegionDammamGCP {
		return t.hyperscalerName + "_" + BTPRegionDammamGCP
	}
	return t.hyperscalerName
}
