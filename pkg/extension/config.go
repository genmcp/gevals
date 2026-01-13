package extension

type ExtensionSpec struct {
	Package string            `json:"package"`
	Env     map[string]string `json:"env,omitempty"`
	Config  map[string]any    `json:"config,omitempty"`
}
