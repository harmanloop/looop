package looop

// MessageKind represents an IPC message
type MessageKind struct {
	Kind uint32
}

// Dispatch - dispatches messages to the appropriate handler
func (m *MessageKind) Dispatch() error {
	return nil
}
