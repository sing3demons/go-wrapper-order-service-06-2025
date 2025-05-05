package logger

type MockLogger struct {
	MethodsToCall map[string]bool
}

func (m *MockLogger) Debugf(format string, args ...any) {
	m.MethodsToCall["Debugf"] = true
}

func (m *MockLogger) Debug(args ...any) {
	m.MethodsToCall["Debug"] = true
}
func (m *MockLogger) Logf(format string, args ...any) {
	m.MethodsToCall["Logf"] = true
}
func (m *MockLogger) Log(args ...any) {
	m.MethodsToCall["Log"] = true
}
func (m *MockLogger) Errorf(format string, args ...any) {
	m.MethodsToCall["Errorf"] = true
}
func (m *MockLogger) Error(args ...any) {
	m.MethodsToCall["Error"] = true
}
func (m *MockLogger) Sync() error {
	m.MethodsToCall["Sync"] = true
	return nil
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		MethodsToCall: make(map[string]bool),
	}
}
