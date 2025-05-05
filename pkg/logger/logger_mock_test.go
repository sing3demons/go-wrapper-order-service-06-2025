package logger

import "testing"

func TestMockLogger(t *testing.T) {
	mockLogger := NewMockLogger()

	// Call all methods
	mockLogger.Debugf("Debugf")
	mockLogger.Debug("Debug")
	mockLogger.Logf("Logf")
	mockLogger.Log("Log")
	mockLogger.Errorf("Errorf")
	mockLogger.Error("Error")
	mockLogger.Sync()

	// Check if all methods were called
	if !mockLogger.MethodsToCall["Debugf"] {
		t.Error("Debugf was not called")
	}
	if !mockLogger.MethodsToCall["Debug"] {
		t.Error("Debug was not called")
	}
	if !mockLogger.MethodsToCall["Logf"] {
		t.Error("Logf was not called")
	}
	if !mockLogger.MethodsToCall["Log"] {
		t.Error("Log was not called")
	}
	if !mockLogger.MethodsToCall["Errorf"] {
		t.Error("Errorf was not called")
	}
	if !mockLogger.MethodsToCall["Error"] {
		t.Error("Error was not called")
	}
	if !mockLogger.MethodsToCall["Sync"] {
		t.Error("Sync was not called")
	}
}
