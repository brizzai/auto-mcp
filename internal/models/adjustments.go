package models

type RouteFieldUpdate struct {
	Method         string `yaml:"method"`
	NewDescription string `yaml:"new_description"`
}

type RouteDescription struct {
	Path    string             `yaml:"path"`
	Updates []RouteFieldUpdate `yaml:"updates"`
}

type RouteSelection struct {
	Path    string   `yaml:"path"`
	Methods []string `yaml:"methods"`
}

type MCPAdjustments struct {
	Descriptions []RouteDescription `yaml:"descriptions,omitempty"`
	Routes       []RouteSelection   `yaml:"routes,omitempty"`
}
