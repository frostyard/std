package reporter

// NoopReporter silently discards all output. Useful for tests and contexts
// where no progress reporting is needed.
type NoopReporter struct{}

func (NoopReporter) Step(int, int, string)       {}
func (NoopReporter) Progress(int, string)        {}
func (NoopReporter) Message(string, ...any)      {}
func (NoopReporter) MessagePlain(string, ...any) {}
func (NoopReporter) Warning(string, ...any)      {}
func (NoopReporter) Error(error, string)         {}
func (NoopReporter) Complete(string, any)        {}
func (NoopReporter) IsJSON() bool                { return false }
