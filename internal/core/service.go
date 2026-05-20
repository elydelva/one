package core

// ServiceID uniquely identifies a service in the catalog (e.g. "github", "notion").
type ServiceID string

// Service is the catalog entry for a third-party API.
type Service struct {
	ID          ServiceID
	Name        string
	Description string
	Version     string
	Actions     []Action
}
