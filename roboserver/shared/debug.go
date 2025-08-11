// Package shared provides debugging and development utilities for the Robomesh server.
//
// This file contains debug functions that provide detailed location information
// for troubleshooting and development. Debug output includes file names, line numbers,
// function names, and call stacks to help identify issues during development.
//
// Debug Mode:
// All debug functions check DEBUG_MODE before producing output.
// Set DEBUG environment variable to "true" to enable debug logging.
//
// Features:
// - Automatic caller detection using runtime.Caller()
// - Clean function name extraction
// - Package-aware formatting
// - Conditional panic behavior for development vs production
// - Color-coded output for different log levels

// shared/debug.go
package shared

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"

	// Bold colors
	ColorBoldRed    = "\033[1;31m"
	ColorBoldGreen  = "\033[1;32m"
	ColorBoldYellow = "\033[1;33m"
	ColorBoldBlue   = "\033[1;34m"
	ColorBoldPurple = "\033[1;35m"
	ColorBoldCyan   = "\033[1;36m"
	ColorBoldWhite  = "\033[1;37m"
)

// TempDebugPrint can be used for temporary debug messages that include file/line info.
func TempDebugPrint(format string, args ...interface{}) {
	if !DEBUG_MODE {
		return
	}

	// Use runtime.Caller(1) to get the caller of TempDebugPrint
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf(ColorPurple+"TEMP DEBUG: "+format+ColorReset+"\n", args...)
		return
	}

	// Get just the filename (not full path)
	filename := filepath.Base(file)

	// Get function name
	funcName := runtime.FuncForPC(pc).Name()
	funcName = getShortFuncName(funcName)

	log.Printf(ColorPurple+"TEMP [%s:%d %s]: "+format+ColorReset+"\n", append([]interface{}{filename, line, funcName}, args...)...)
}

// DebugPrint automatically gets file, line, and function info
func DebugPrint(format string, args ...interface{}) {
	if !DEBUG_MODE {
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugPrint
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf(ColorCyan+"DEBUG: "+format+ColorReset+"\n", args...)
		return
	}

	// Get just the filename (not full path)
	filename := filepath.Base(file)

	// Get function name
	funcName := runtime.FuncForPC(pc).Name()
	funcName = getShortFuncName(funcName)

	// Format: [filename:line funcName] message
	log.Printf(ColorCyan+"[%s:%d %s]: "+format+ColorReset+"\n", append([]interface{}{filename, line, funcName}, args...)...)
}

// DebugError prints an error message with file/line info
func DebugError(err error) {
	if !DEBUG_MODE {
		log.Printf(ColorRed+"ERROR: %v"+ColorReset+"\n", err)
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugError
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf(ColorRed+"ERROR: %v"+ColorReset+"\n", err)
		return
	}

	filename := filepath.Base(file)
	funcName := getShortFuncName(runtime.FuncForPC(pc).Name())

	log.Printf(ColorRed+"ERROR [%s:%d %s]: %v"+ColorReset+"\n", filename, line, funcName, err)
}

func DebugErrorf(format string, args ...interface{}) {
	if !DEBUG_MODE {
		log.Printf(ColorRed+"ERROR: "+format+ColorReset+"\n", args...)
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugErrorf
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf(ColorRed+"ERROR: "+format+ColorReset+"\n", args...)
		return
	}

	filename := filepath.Base(file)
	funcName := getShortFuncName(runtime.FuncForPC(pc).Name())

	log.Printf(ColorRed+"ERROR [%s:%d %s]: "+format+ColorReset+"\n", append([]interface{}{filename, line, funcName}, args...)...)
}

// DebugPrintWithPackage shows package/file:line format
func DebugPrintWithPackage(format string, args ...interface{}) {
	if !DEBUG_MODE {
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugPrintWithPackage
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf(ColorBlue+"DEBUG: "+format+ColorReset, args...)
		return
	}

	// Get package and file
	packagePath := getPackageFromFile(file)
	filename := filepath.Base(file)
	funcName := getShortFuncName(runtime.FuncForPC(pc).Name())

	log.Printf(ColorBlue+"[%s/%s:%d %s]: "+format+ColorReset,
		append([]interface{}{packagePath, filename, line, funcName}, args...)...)
}

func DebugPanic(format string, args ...interface{}) {
	if !DEBUG_MODE {
		log.Printf(ColorBoldRed+"CRITICAL ERROR (would panic in debug): "+format+ColorReset, args...)
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugPanic
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Panicf(ColorBoldRed+"PANIC: "+format+ColorReset, args...)
		return
	}

	filename := filepath.Base(file)
	funcName := getShortFuncName(runtime.FuncForPC(pc).Name())

	log.Panicf(ColorBoldRed+"PANIC [%s:%d %s]: "+format+ColorReset,
		append([]interface{}{filename, line, funcName}, args...)...)
}

// Remove the redundant DebugPrintln - it's causing double wrapping
// Instead, users can just add \n to their format string if needed

// Helper to extract package name from file path
func getPackageFromFile(file string) string {
	dir := filepath.Dir(file)
	return filepath.Base(dir)
}

// Helper to get short function name
func getShortFuncName(fullName string) string {
	// Remove package path
	if lastSlash := strings.LastIndex(fullName, "/"); lastSlash >= 0 {
		fullName = fullName[lastSlash+1:]
	}
	// Remove receiver/package prefix, keep just function name
	if lastDot := strings.LastIndex(fullName, "."); lastDot >= 0 {
		return fullName[lastDot+1:]
	}
	return fullName
}
