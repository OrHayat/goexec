package goexec

// Command represents a command to be executed
type Command struct {
	// Cmd is the command string to run
	Cmd string
	// Args are the command arguments
	Args []string
	// HighPriority determines whether this command should try
	// to acquire a high-priority slot in the backend
	HighPriority bool
}
