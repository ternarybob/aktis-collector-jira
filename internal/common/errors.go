package common

import (
	"fmt"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeConfiguration for configuration-related errors
	ErrorTypeConfiguration ErrorType = "configuration"
	// ErrorTypeValidation for validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeService for service-level errors
	ErrorTypeService ErrorType = "service"
	// ErrorTypeStorage for storage/persistence errors
	ErrorTypeStorage ErrorType = "storage"
	// ErrorTypeNetwork for network-related errors
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeAuth for authentication/authorization errors
	ErrorTypeAuth ErrorType = "auth"
	// ErrorTypeJira for Jira-specific errors
	ErrorTypeJira ErrorType = "jira"
	// ErrorTypeCollection for data collection errors
	ErrorTypeCollection ErrorType = "collection"
	// ErrorTypeInternal for internal system errors
	ErrorTypeInternal ErrorType = "internal"
)

// CollectorError represents a structured error with context
type CollectorError struct {
	Type      ErrorType              `json:"type"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Cause     error                  `json:"-"`
}

// Error implements the error interface
func (e *CollectorError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s:%s] %s: %s", e.Type, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

// Unwrap implements the errors.Unwrap interface
func (e *CollectorError) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error
func (e *CollectorError) WithContext(key string, value interface{}) *CollectorError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithCause sets the underlying cause
func (e *CollectorError) WithCause(cause error) *CollectorError {
	e.Cause = cause
	return e
}

// NewError creates a new CollectorError
func NewError(errorType ErrorType, code, message string) *CollectorError {
	return &CollectorError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(code, message string) *CollectorError {
	return NewError(ErrorTypeConfiguration, code, message)
}

// NewValidationError creates a validation error
func NewValidationError(code, message string) *CollectorError {
	return NewError(ErrorTypeValidation, code, message)
}

// NewServiceError creates a service error
func NewServiceError(code, message string) *CollectorError {
	return NewError(ErrorTypeService, code, message)
}

// NewStorageError creates a storage error
func NewStorageError(code, message string) *CollectorError {
	return NewError(ErrorTypeStorage, code, message)
}

// NewNetworkError creates a network error
func NewNetworkError(code, message string) *CollectorError {
	return NewError(ErrorTypeNetwork, code, message)
}

// NewAuthError creates an authentication error
func NewAuthError(code, message string) *CollectorError {
	return NewError(ErrorTypeAuth, code, message)
}

// NewJiraError creates a Jira-specific error
func NewJiraError(code, message string) *CollectorError {
	return NewError(ErrorTypeJira, code, message)
}

// NewCollectionError creates a collection error
func NewCollectionError(code, message string) *CollectorError {
	return NewError(ErrorTypeCollection, code, message)
}

// NewInternalError creates an internal system error
func NewInternalError(code, message string) *CollectorError {
	return NewError(ErrorTypeInternal, code, message)
}

// WrapError wraps an existing error with CollectorError context
func WrapError(err error, errorType ErrorType, code, message string) *CollectorError {
	return &CollectorError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Cause:     err,
	}
}