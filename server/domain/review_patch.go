package domain

// ReviewPatch is the agent's change to the review of one source. For a source with no review
// it carries every candidate. For an existing review it carries only the candidates to add or
// change; candidates it leaves out stay as they are.
type ReviewPatch struct {
	SourceID string
	// ExpectedRevision is the review revision the agent read. A newer revision rejects the patch.
	ExpectedRevision int
	Summary          string
	Notification     string
	// Candidates without an ID are new. Candidates with an ID update the queued, pending or
	// open candidate of that ID, and their Revision must be the one the agent read.
	Candidates []CaptureCandidate
}
