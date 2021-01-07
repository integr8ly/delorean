package types

const (
	OlmTypeRhmi  = "integreatly-operator"
	OlmTypeRhoam = "managed-api-service"
)

type RhssoManifest struct {
	Spec RhssoManifestSpec `yaml:"spec"`
}

type RhssoManifestSpec struct {
	Install RhssoManifestInstall `yaml:"install"`
}

type RhssoManifestInstall struct {
	Spec RhssoManifestInstallSpec `yaml:"spec"`
}
type RhssoManifestInstallSpec struct {
	Deployments []RhssoManifestDeployments `yaml:"deployments"`
}

type RhssoManifestDeployments struct {
	Name string                       `yaml:"name"`
	Spec RhssoManifestDeploymentsSpec `yaml:"spec"`
}

type RhssoManifestDeploymentsSpec struct {
	Template RhssoManifestTemplate `yaml:"template"`
}

type RhssoManifestTemplate struct {
	Spec RhssoManifestTemplateSpec `yaml:"spec"`
}

type RhssoManifestTemplateSpec struct {
	Containers []RhssoManifestDeploymentsContainers `yaml:"containers"`
}

type RhssoManifestDeploymentsContainers struct {
	Env []EnvType `yaml:"env"`
}

type EnvType struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type RhssoPackage struct {
	Channels []RhssoPackageChannels `yaml:"channels"`
}

type RhssoPackageChannels struct {
	Name       string `yaml:"name"`
	CurrentCSV string `yaml:"currentCSV"`
}
