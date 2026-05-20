package core

// Inputs holds the user-provided arguments for an action, keyed by parameter name.
type Inputs map[string]any

// Get returns the value for a key, or nil if absent.
func (i Inputs) Get(key string) any {
	if i == nil {
		return nil
	}
	return i[key]
}
