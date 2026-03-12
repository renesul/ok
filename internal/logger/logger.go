package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	logLevelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}

	currentLevel = INFO
	mu           sync.RWMutex

	// Ring buffer for recent log entries, queryable via /logs endpoint.
	recentLogs    [512]LogEntry
	recentHead    int
	recentCount   int
	recentTotal   int // monotonic counter of all entries ever written
	recentMu      sync.Mutex

	// File-based logging
	logDir  string
	fileMu  sync.Mutex
	files   map[string]*os.File // sanitized component name → file handle
	allFile *os.File
)

type LogEntry struct {
	Level     string         `json:"level"`
	Timestamp string         `json:"timestamp"`
	Component string         `json:"component,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Caller    string         `json:"caller,omitempty"`
}

func SetLevel(level LogLevel) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

func GetLevel() LogLevel {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

func logMessage(level LogLevel, component string, message string, fields map[string]any) {
	if level < currentLevel {
		return
	}

	entry := LogEntry{
		Level:     logLevelNames[level],
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Component: component,
		Message:   message,
		Fields:    fields,
	}

	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			entry.Caller = fmt.Sprintf("%s:%d (%s)", file, line, fn.Name())
		}
	}

	// Store in ring buffer
	recentMu.Lock()
	recentLogs[recentHead] = entry
	recentHead = (recentHead + 1) % len(recentLogs)
	if recentCount < len(recentLogs) {
		recentCount++
	}
	recentTotal++
	recentMu.Unlock()

	// Write to per-component and all.log files
	writeToFiles(entry, component)

	// Output JSONL to stdout for structured capture by launcher
	if jsonData, err := json.Marshal(entry); err == nil {
		log.Println(string(jsonData))
	} else {
		var fieldStr string
		if len(fields) > 0 {
			fieldStr = " " + formatFields(fields)
		}
		log.Printf("[%s] [%s]%s %s%s",
			entry.Timestamp, logLevelNames[level],
			formatComponent(component), message, fieldStr)
	}

	if level == FATAL {
		os.Exit(1)
	}
}

// RecentEntries returns log entries written after the given offset (monotonic index).
// Returns the entries and the current total count (to be used as next offset).
func RecentEntries(afterOffset int) ([]LogEntry, int) {
	recentMu.Lock()
	defer recentMu.Unlock()

	if afterOffset >= recentTotal {
		return nil, recentTotal
	}

	// How many new entries since afterOffset
	want := recentTotal - afterOffset
	if want > recentCount {
		want = recentCount
	}

	entries := make([]LogEntry, want)
	start := (recentHead - want + len(recentLogs)) % len(recentLogs)
	for i := range want {
		entries[i] = recentLogs[(start+i)%len(recentLogs)]
	}
	return entries, recentTotal
}

func formatComponent(component string) string {
	if component == "" {
		return ""
	}
	return fmt.Sprintf(" %s:", component)
}

func formatFields(fields map[string]any) string {
	parts := make([]string, 0, len(fields))
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

func DebugC(component string, message string) {
	logMessage(DEBUG, component, message, nil)
}

func DebugCF(component string, message string, fields map[string]any) {
	logMessage(DEBUG, component, message, fields)
}

func InfoC(component string, message string) {
	logMessage(INFO, component, message, nil)
}

func InfoCF(component string, message string, fields map[string]any) {
	logMessage(INFO, component, message, fields)
}

func WarnC(component string, message string) {
	logMessage(WARN, component, message, nil)
}

func WarnCF(component string, message string, fields map[string]any) {
	logMessage(WARN, component, message, fields)
}

func ErrorC(component string, message string) {
	logMessage(ERROR, component, message, nil)
}

func ErrorCF(component string, message string, fields map[string]any) {
	logMessage(ERROR, component, message, fields)
}

func FatalCF(component string, message string, fields map[string]any) {
	logMessage(FATAL, component, message, fields)
}

// InitFileLogging enables JSONL file logging. Creates dir and opens all.log.
// Each component gets its own file lazily on first log entry.
func InitFileLogging(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "all.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	fileMu.Lock()
	logDir = dir
	allFile = f
	files = make(map[string]*os.File)
	fileMu.Unlock()
	return nil
}

// CloseFileLogging closes all open log file handles.
func CloseFileLogging() {
	fileMu.Lock()
	defer fileMu.Unlock()
	for _, f := range files {
		f.Close()
	}
	files = nil
	if allFile != nil {
		allFile.Close()
		allFile = nil
	}
	logDir = ""
}

// LogDir returns the current log directory path (empty if file logging is disabled).
func LogDir() string {
	fileMu.Lock()
	defer fileMu.Unlock()
	return logDir
}

// LogComponents returns sorted list of component names that have log files.
// "all" is always first if it exists.
func LogComponents() []string {
	fileMu.Lock()
	dir := logDir
	fileMu.Unlock()
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var components []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".log")
		if name == "all" {
			continue
		}
		components = append(components, name)
	}
	sort.Strings(components)
	return append([]string{"all"}, components...)
}

func sanitizeComponent(component string) string {
	if component == "" {
		return "general"
	}
	return strings.ReplaceAll(component, ".", "_")
}

// getComponentFile returns the file handle for a component, creating it if needed.
// Must be called with fileMu held.
func getComponentFile(component string) *os.File {
	name := sanitizeComponent(component)
	if f, ok := files[name]; ok {
		return f
	}
	f, err := os.OpenFile(filepath.Join(logDir, name+".log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil
	}
	files[name] = f
	return f
}

func writeToFiles(entry LogEntry, component string) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')
	fileMu.Lock()
	defer fileMu.Unlock()
	if logDir == "" {
		return
	}
	if f := getComponentFile(component); f != nil {
		f.Write(data)
	}
	if allFile != nil {
		allFile.Write(data)
	}
}
