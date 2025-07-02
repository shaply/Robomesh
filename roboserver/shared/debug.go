// shared/debug.go
package shared

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"
)

// DebugPrint automatically gets file, line, and function info
func DebugPrint(format string, args ...interface{}) {
	if !DEBUG_MODE {
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugPrint
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf("DEBUG: "+format+"\n", args...)
		return
	}

	// Get just the filename (not full path)
	filename := filepath.Base(file)

	// Get function name
	funcName := runtime.FuncForPC(pc).Name()
	funcName = getShortFuncName(funcName)

	// Format: [filename:line funcName] message
	log.Printf("[%s:%d %s]: "+format+"\n", append([]interface{}{filename, line, funcName}, args...)...)
}

// DebugPrintWithPackage shows package/file:line format
func DebugPrintWithPackage(format string, args ...interface{}) {
	if !DEBUG_MODE {
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugPrintWithPackage
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf("DEBUG: "+format, args...)
		return
	}

	// Get package and file
	packagePath := getPackageFromFile(file)
	filename := filepath.Base(file)
	funcName := getShortFuncName(runtime.FuncForPC(pc).Name())

	log.Printf("[%s/%s:%d %s]: "+format,
		append([]interface{}{packagePath, filename, line, funcName}, args...)...)
}

func DebugPanic(format string, args ...interface{}) {
	if !DEBUG_MODE {
		log.Printf("CRITICAL ERROR (would panic in debug): "+format, args...)
		return
	}

	// Use runtime.Caller(1) to get the caller of DebugPanic
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Panicf("PANIC: "+format, args...)
		return
	}

	filename := filepath.Base(file)
	funcName := getShortFuncName(runtime.FuncForPC(pc).Name())

	log.Panicf("PANIC [%s:%d %s]: "+format,
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
