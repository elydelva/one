package core

// InstallGuide is a setup document for a service (e.g. "share a Notion page").
type InstallGuide struct {
	ID      string
	Service ServiceID
	Title   string
	Content string
	Verify  *VerifyStep
}

// VerifyStep is an optional action to run to confirm setup succeeded.
type VerifyStep struct {
	Action ActionID
	Hint   string
}
