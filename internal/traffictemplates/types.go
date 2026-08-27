package traffictemplates

type Param struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
	Label string `yaml:"label,omitempty"`
}

type Template struct {
	ID          string  `yaml:"id"`
	BundledSlug string  `yaml:"bundled_slug,omitempty"`
	Name        string  `yaml:"name"`
	Category    string  `yaml:"category"`
	CostSync    string  `yaml:"cost_sync,omitempty"`
	Notes       string  `yaml:"notes,omitempty"`
	Params      []Param `yaml:"params"`
	Generated   bool    `yaml:"-"`
}

type SidecarFile struct {
	Version   int        `yaml:"version"`
	Templates []Template `yaml:"templates"`
}
