// globals defines the global variables shared between primary and backup.
package globals

var (
	// The Operation request ID.
	OpNum int

	// The current view number.
	ViewNum int

	// The current commit number.
	CommitNum int

	AllPorts = map[int]int{0: 1234, 1: 9000, 2: 9001, 3: 9002, 4: 9003}
)
