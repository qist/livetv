package global

import (
	"strings"
	"sync"
)

// LogStream keeps a rolling buffer of log lines and allows tail-style streaming.
type LogStream struct {
	mu      sync.Mutex
	lines   []string
	max     int
	subs    map[chan string]struct{}
	partial string
}

func NewLogStream(max int) *LogStream {
	if max <= 0 {
		max = 1000
	}
	return &LogStream{
		max:  max,
		subs: make(map[chan string]struct{}),
	}
}

// Write implements io.Writer and splits on newlines to store log lines.
func (ls *LogStream) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	ls.mu.Lock()
	data := ls.partial + string(p)
	parts := strings.Split(data, "\n")
	ls.partial = parts[len(parts)-1]
	lines := parts[:len(parts)-1]
	for _, line := range lines {
		ls.lines = append(ls.lines, line)
		if len(ls.lines) > ls.max {
			ls.lines = ls.lines[len(ls.lines)-ls.max:]
		}
	}
	subs := make([]chan string, 0, len(ls.subs))
	for ch := range ls.subs {
		subs = append(subs, ch)
	}
	ls.mu.Unlock()

	for _, line := range lines {
		for _, ch := range subs {
			select {
			case ch <- line:
			default:
			}
		}
	}
	return len(p), nil
}

// Snapshot returns the current buffered lines.
func (ls *LogStream) Snapshot() []string {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	out := make([]string, len(ls.lines))
	copy(out, ls.lines)
	return out
}

// Subscribe returns a channel that receives new log lines.
// The cancel function removes the subscription.
func (ls *LogStream) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 100)
	ls.mu.Lock()
	ls.subs[ch] = struct{}{}
	ls.mu.Unlock()

	cancel := func() {
		ls.mu.Lock()
		delete(ls.subs, ch)
		ls.mu.Unlock()
	}
	return ch, cancel
}

// LogStreamBuffer holds recent logs for /log streaming.
var LogStreamBuffer = NewLogStream(2000)

// LogFilePath is the configured log file path (may not exist).
var LogFilePath string
