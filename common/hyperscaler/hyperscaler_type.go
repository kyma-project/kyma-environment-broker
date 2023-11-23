package hyperscaler

type Type struct {
	hyperscalerName   string
	hyperscalerRegion string
}

func GCP() Type {
	return Type{
		hyperscalerName: "gcp",
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

func Openstack(region string) Type {
	return Type{
		hyperscalerName:   "openstack",
		hyperscalerRegion: region,
	}
}

func (t Type) GetName() string {
	return t.hyperscalerName
}

func (t Type) SetRegion(region string) {
	t.hyperscalerName = region
}

func (t Type) GetKey() string {
	if t.hyperscalerName == "openstack" && t.hyperscalerRegion != "" {
		return t.hyperscalerName + "_" + t.hyperscalerRegion
	}
	return t.hyperscalerName
}
