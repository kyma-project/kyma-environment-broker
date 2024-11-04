package orchestration

type Config struct {
	KubernetesVersion string `envconfig:"-"`
	Namespace         string
	Name              string
}
