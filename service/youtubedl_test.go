package service

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeYoutubeURL(t *testing.T) {
	input := " \t`\"https://www.youtube.com/watch?v=abc123\"`\n"
	got := normalizeYoutubeURL(input)
	want := "https://www.youtube.com/watch?v=abc123"
	if got != want {
		t.Fatalf("normalizeYoutubeURL() = %q, want %q", got, want)
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
