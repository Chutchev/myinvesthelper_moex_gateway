package apperrors

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrHTTPError      = errors.New("HTTP request failed")
	ErrTimeout        = errors.New("request timeout")
	ErrParseError     = errors.New("failed to parse response")
	ErrCacheMiss      = errors.New("cache miss")
	ErrCacheError     = errors.New("cache operation failed")
)
