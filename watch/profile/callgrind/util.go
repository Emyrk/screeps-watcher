package callgrind

func get(m map[string]string, key, defaultValue string) string {
	if v, found := m[key]; found {
		return v
	}
	return defaultValue
}
