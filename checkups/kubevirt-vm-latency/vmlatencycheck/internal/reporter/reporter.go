package reporter

type configMapReporter struct {
	client             interface{}
	configMapName      string
	configMapNamespace string
}

func New(client interface{}, namespace, name string) *configMapReporter {
	return &configMapReporter{
		client:             client,
		configMapNamespace: namespace,
		configMapName:      name,
	}
}

func (r *configMapReporter) Report(_ map[string]string) error {
	return nil
}
