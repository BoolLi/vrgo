// globals defines the global variables shared between primary and backup.
package globals

var (
	// The Operation request ID.
	OpNum int

	// The current view number.
	ViewNum int

	// The current commit number.
	CommitNum int
)
