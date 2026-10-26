package service

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeYoutubeURL(t *testing.T) {
	input := " \t`\"https://www.youtube.com/watch?v=abc123\"`\n"
	got := NormalizeYoutubeURL(input)
	want := "https://www.youtube.com/watch?v=abc123"
	if got != want {
		t.Fatalf("NormalizeYoutubeURL() = %q, want %q", got, want)
	}
}

func TestSplitArgsKeepsQuotedExtractorArgs(t *testing.T) {
	args := splitArgs(`--js-runtimes deno --extractor-args 'youtube:player-client=web' --add-header "User-Agent: Test UA" -f b -g {url}`)
	if len(args) != 10 {
		t.Fatalf("splitArgs() returned %d args, want 10: %#v", len(args), args)
	}
	if args[2] != "--extractor-args" || args[3] != "youtube:player-client=web" {
		t.Fatalf("splitArgs() did not preserve extractor args: %#v", args)
	}
	if args[4] != "--add-header" || args[5] != "User-Agent: Test UA" {
		t.Fatalf("splitArgs() did not preserve header arg: %#v", args)
	}
}

func TestBuildYtdlErrorForBotCheck(t *testing.T) {
	err := buildYtdlError(errors.New("exit status 1"), []byte("ERROR: Sign in to confirm you're not a bot"))
	if err == nil {
		t.Fatal("buildYtdlError() returned nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cookies or visitor data") {
		t.Fatalf("buildYtdlError() = %q, want cookies hint", err.Error())
	}
}

func TestShouldRetryWithAndroidClient(t *testing.T) {
	formatErr := []byte("ERROR: [youtube] abc123: Requested format is not available. Use --list-formats for a list of available formats")
	cases := []struct {
		name   string
		args   []string
		stderr []byte
		want   bool
	}{
		{"format error without player-client", []string{"-f", "b", "-g", "{url}"}, formatErr, true},
		{"no formats without player-client", []string{"-f", "b", "-g", "{url}"}, []byte("ERROR: No video formats found!"), true},
		{"player-client already configured", []string{"--extractor-args", "youtube:player-client=tv", "-f", "b"}, formatErr, false},
		{"player-client in equals form", []string{"--extractor-args=youtube:player-client=tv", "-f", "b"}, formatErr, false},
		{"player_client in underscore form", []string{"--extractor-args", "youtube:player_client=tv", "-f", "b"}, formatErr, false},
		{"unrelated error", []string{"-f", "b"}, []byte("ERROR: Video unavailable"), false},
	}
	for _, tc := range cases {
		if got := shouldRetryWithAndroidClient(tc.args, tc.stderr); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestWithAndroidClientArgs(t *testing.T) {
	got := withAndroidClientArgs([]string{"-f", "b", "-g", "https://example.com/watch?v=x"})
	want := []string{"--extractor-args", "youtube:player-client=android", "-f", "b", "-g", "https://example.com/watch?v=x"}
	if len(got) != len(want) {
		t.Fatalf("withAndroidClientArgs() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("withAndroidClientArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
