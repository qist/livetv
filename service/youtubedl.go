package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/qist/livetv/global"
)

type ytdlCall struct {
	done    chan struct{}
	liveURL string
	err     error
}

type ytdlFailure struct {
	err   error
	until time.Time
}

var (
	ytdlCallMu        sync.Mutex
	ytdlInflightCalls = make(map[string]*ytdlCall)
	ytdlFailures      = make(map[string]ytdlFailure)
)

const ytdlFailureBackoff = time.Minute

func GetYoutubeLiveM3U8(youtubeURL string) (string, error) {
	normalizedURL := NormalizeYoutubeURL(youtubeURL)
	if normalizedURL == "" {
		return "", fmt.Errorf("empty youtube url")
	}
	if liveURL, ok := global.URLCache.Load(normalizedURL); ok {
		return liveURL.(string), nil
	}
	if cachedErr := getYtdlFailure(normalizedURL); cachedErr != nil {
		log.Println("skip yt-dlp retry during backoff", normalizedURL)
		return "", cachedErr
	}

	call, waiting := getOrCreateYtdlCall(normalizedURL)
	if waiting {
		<-call.done
		return call.liveURL, call.err
	}
	defer finishYtdlCall(normalizedURL, call)

	if liveURL, ok := global.URLCache.Load(normalizedURL); ok {
		call.liveURL = liveURL.(string)
		return call.liveURL, nil
	}

	log.Println("cache miss", normalizedURL)
	call.liveURL, call.err = RealGetYoutubeLiveM3U8(normalizedURL)
	if call.err != nil {
		log.Println(call.err)
		log.Println("[YTDL]", call.liveURL)
		if shouldBackoffYtdl(call.err) {
			setYtdlFailure(normalizedURL, call.err)
		} else {
			clearYtdlFailure(normalizedURL)
		}
		return "", call.err
	}
	clearYtdlFailure(normalizedURL)
	global.URLCache.Store(normalizedURL, call.liveURL)
	return call.liveURL, nil
}

func RealGetYoutubeLiveM3U8(youtubeURL string) (string, error) {
	youtubeURL = NormalizeYoutubeURL(youtubeURL)
	cc := GetCachedConfig()
	YtdlCmd := strings.TrimSpace(cc.YtdlCmd.Load().(string))
	if YtdlCmd == "" {
		YtdlCmd = "yt-dlp"
	}
	YtdlArgs := cc.YtdlArgs.Load().(string)
	YtdlCookies := strings.TrimSpace(cc.YtdlCookies.Load().(string))
	ytdlArgs := splitArgs(strings.TrimSpace(YtdlArgs))
	hasURLArg := false
	for i, v := range ytdlArgs {
		if strings.EqualFold(v, "{url}") {
			ytdlArgs[i] = youtubeURL
			hasURLArg = true
		}
	}
	if !hasURLArg {
		ytdlArgs = append(ytdlArgs, youtubeURL)
	}
	ytdlCookies := strings.TrimSpace(YtdlCookies)
	if ytdlCookies != "" {
		if !filepath.IsAbs(ytdlCookies) {
			datadir := os.Getenv("LIVETV_DATADIR")
			if datadir == "" {
				datadir = "./data"
			}
			ytdlCookies = filepath.Join(datadir, ytdlCookies)
		}
		ytdlArgs = append(ytdlArgs, "--cookies", ytdlCookies)
	}
	_, err := exec.LookPath(YtdlCmd)
	if err != nil {
		log.Println(err)
		return "", err
	}
	timeout := GetYtdlTimeout()
	stdout, stderr, err := runYtdlCommand(YtdlCmd, ytdlArgs, timeout)
	if err != nil && shouldRetryWithAndroidClient(ytdlArgs, stderr) {
		retryArgs := withAndroidClientArgs(ytdlArgs)
		log.Println("[YTDL] requested format unavailable, retrying with android client")
		stdout, stderr, err = runYtdlCommand(YtdlCmd, retryArgs, timeout)
	}
	if err != nil {
		if len(stderr) > 0 {
			log.Println("[YTDL stderr]", string(stderr))
		}
		wrappedErr := err
		if parsedErr := buildYtdlError(err, stderr); parsedErr != nil {
			wrappedErr = parsedErr
		}
		log.Println(wrappedErr)
		return "", wrappedErr
	}
	if len(stderr) > 0 {
		log.Println("[YTDL stderr]", string(stderr))
	}
	return strings.TrimSpace(string(stdout)), nil
}

func runYtdlCommand(cmdName string, args []string, timeout time.Duration) ([]byte, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdName, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	log.Println(cmdName, args)
	runErr := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		if stderr.Len() > 0 {
			log.Println("[YTDL stderr]", stderr.String())
		}
		log.Println("[YTDL] timeout after", timeout)
		return stdout.Bytes(), stderr.Bytes(), ctx.Err()
	}
	return stdout.Bytes(), stderr.Bytes(), runErr
}

// shouldRetryWithAndroidClient reports whether yt-dlp failed only because the
// default player clients returned no muxed (video+audio) format. For live
// streams the tv/visionos clients often expose separate video/audio tracks,
// so "-f b" cannot match anything; the android client still provides muxed
// HLS formats. Skip the retry when the user already configured a client.
func shouldRetryWithAndroidClient(args []string, stderr []byte) bool {
	if hasPlayerClientArg(args) {
		return false
	}
	text := strings.ToLower(string(stderr))
	return strings.Contains(text, "requested format is not available") ||
		strings.Contains(text, "no video formats found")
}

func withAndroidClientArgs(args []string) []string {
	retryArgs := make([]string, 0, len(args)+2)
	retryArgs = append(retryArgs, "--extractor-args", "youtube:player-client=android")
	return append(retryArgs, args...)
}

func hasPlayerClientArg(args []string) bool {
	for _, arg := range args {
		lower := strings.ToLower(arg)
		if strings.Contains(lower, "player-client") || strings.Contains(lower, "player_client") {
			return true
		}
	}
	return false
}

func getOrCreateYtdlCall(cacheKey string) (*ytdlCall, bool) {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	if call, ok := ytdlInflightCalls[cacheKey]; ok {
		return call, true
	}
	call := &ytdlCall{done: make(chan struct{})}
	ytdlInflightCalls[cacheKey] = call
	return call, false
}

func finishYtdlCall(cacheKey string, call *ytdlCall) {
	ytdlCallMu.Lock()
	delete(ytdlInflightCalls, cacheKey)
	ytdlCallMu.Unlock()
	close(call.done)
}

func getYtdlFailure(cacheKey string) error {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	failure, ok := ytdlFailures[cacheKey]
	if !ok {
		return nil
	}
	if time.Now().After(failure.until) {
		delete(ytdlFailures, cacheKey)
		return nil
	}
	return failure.err
}

func setYtdlFailure(cacheKey string, err error) {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	ytdlFailures[cacheKey] = ytdlFailure{
		err:   fmt.Errorf("yt-dlp temporary backoff for 1m: %w", err),
		until: time.Now().Add(ytdlFailureBackoff),
	}
}

func clearYtdlFailure(cacheKey string) {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	delete(ytdlFailures, cacheKey)
}

func resetYtdlFailureState() {
	ytdlCallMu.Lock()
	defer ytdlCallMu.Unlock()
	ytdlInflightCalls = make(map[string]*ytdlCall)
	ytdlFailures = make(map[string]ytdlFailure)
}

func shouldBackoffYtdl(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "http error 429") ||
		strings.Contains(message, "too many requests") ||
		strings.Contains(message, "sign in to confirm") ||
		strings.Contains(message, "not a bot") ||
		strings.Contains(message, "visitor data")
}

func buildYtdlError(waitErr error, stderrBytes []byte) error {
	stderrText := strings.TrimSpace(string(stderrBytes))
	if stderrText == "" {
		return waitErr
	}
	lowerText := strings.ToLower(stderrText)
	if strings.Contains(lowerText, "sign in to confirm you’re not a bot") ||
		strings.Contains(lowerText, "sign in to confirm you're not a bot") ||
		strings.Contains(lowerText, "http error 429") ||
		strings.Contains(lowerText, "too many requests") {
		return fmt.Errorf("youtube requires cookies or visitor data: %w", waitErr)
	}
	return fmt.Errorf("%w: %s", waitErr, stderrText)
}

func NormalizeYoutubeURL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimFunc(s, func(r rune) bool {
		switch r {
		case '`', '\'', '"', '“', '”', '‘', '’', '<', '>':
			return true
		default:
			return unicode.IsSpace(r)
		}
	})
	return strings.TrimSpace(s)
}

func splitArgs(s string) []string {
	var args []string
	var buf strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	flush := func() {
		if buf.Len() > 0 {
			args = append(args, buf.String())
			buf.Reset()
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			buf.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' && !inSingle {
			escaped = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inSingle && !inDouble {
			flush()
			continue
		}
		buf.WriteByte(ch)
	}
	flush()
	return args
}
