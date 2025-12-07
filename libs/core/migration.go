package core

import (
	"fmt"
	"os"
	"sync"
)

// EncapsulationMode controls enforcement level during migration
type EncapsulationMode int

const (
	// EncapsulationDisabled - No validation (bypass all checks)
	EncapsulationDisabled EncapsulationMode = iota
	// EncapsulationWarn - Log violations, don't error
	EncapsulationWarn
	// EncapsulationEnforce - Error on violations
	EncapsulationEnforce
)

var (
	currentEncapsulationMode     = EncapsulationDisabled
	encapsulationModeMutex       sync.RWMutex
	encapsulationViolationLogger = os.Stderr
)

// SetEncapsulationMode configures enforcement level
func SetEncapsulationMode(mode EncapsulationMode) {
	encapsulationModeMutex.Lock()
	defer encapsulationModeMutex.Unlock()
	currentEncapsulationMode = mode
}

// GetEncapsulationMode returns current enforcement level
func GetEncapsulationMode() EncapsulationMode {
	encapsulationModeMutex.RLock()
	defer encapsulationModeMutex.RUnlock()
	return currentEncapsulationMode
}

// SetEncapsulationViolationLogger sets the output for violation warnings
func SetEncapsulationViolationLogger(w *os.File) {
	encapsulationModeMutex.Lock()
	defer encapsulationModeMutex.Unlock()
	encapsulationViolationLogger = w
}

// CheckEncapsulationViolation checks if access is allowed based on current mode
// Returns (allowed, error) where error is nil if allowed or mode is Warn
func CheckEncapsulationViolation(fromModule, toModule, serviceName string) (bool, error) {
	encapsulationModeMutex.RLock()
	mode := currentEncapsulationMode
	logger := encapsulationViolationLogger
	encapsulationModeMutex.RUnlock()

	if mode == EncapsulationDisabled {
		return true, nil
	}

	errMsg := fmt.Errorf(
		"module '%s' cannot access unexported provider '%s' from module '%s'",
		fromModule,
		serviceName,
		toModule,
	)

	if mode == EncapsulationWarn {
		fmt.Fprintf(logger, "WARNING: %s\n", errMsg)
		return true, nil
	}

	// EncapsulationEnforce
	return false, errMsg
}