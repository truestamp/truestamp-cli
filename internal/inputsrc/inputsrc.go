// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package inputsrc resolves a CLI input to a byte slice (or a stream) from
// one of six conventional sources: a positional argument, an explicit file
// path, an interactive file picker, an explicit URL, an interactive URL
// prompt, or a stdin pipe. The single-dash "-" positional is accepted as an
// alias for stdin, matching Unix convention.
//
// Callers declare their input flags with NoOptDefVal set to [FilePickSentinel]
// or [URLPromptSentinel] so that `--file` or `--url` without a value triggers
// the interactive path. An empty flag value means the flag was not set.
package inputsrc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Sentinel flag values set via pflag's NoOptDefVal. The parenthesised
// form is used so that pflag's help renderer — which prints
// `--file string[="<NoOptDefVal>"]` — produces something readable
// rather than whitespace or control characters. A user typing
// `--file=(pick)` or `--url=(prompt)` will hit the interactive flow,
// which is harmless since those literal filenames/URLs are effectively
// impossible in practice.
const (
	FilePickSentinel  = "(pick)"
	URLPromptSentinel = "(prompt)"
)

// SourceType identifies where the resolved bytes came from.
type SourceType string

const (
	SourceFile   SourceType = "file"
	SourceStdin  SourceType = "stdin"
	SourceURL    SourceType = "url"
	SourcePicker SourceType = "picker"
)

// Source describes where the resolved bytes came from. Path holds a
// filesystem path (for SourceFile/SourcePicker) or a URL (for SourceURL).
// For SourceStdin, Path is "-" to match the Unix convention used by
// tools like sha256sum.
type Source struct {
	Type SourceType
	Path string
	Size int64 // -1 when unknown
}

// DisplayName returns a human-friendly identifier for the source.
// Files and URLs use their path/URL; stdin renders as "-".
func (s Source) DisplayName() string {
	if s.Path != "" {
		return s.Path
	}
	return "-"
}

// Options configures input resolution. Only the subset relevant to the
// caller's flag surface needs to be set; unset fields disable their mode.
type Options struct {
	// PositionalArg is the first positional argument, if any. An empty
	// string means no positional arg was supplied.
	PositionalArg string

	// FileFlag is the value of --file (or equivalent). Empty = unset;
	// FilePickSentinel triggers the interactive file picker; any other
	// non-empty value is treated as a filesystem path.
	FileFlag string

	// URLFlag is the value of --url (or equivalent). Empty = unset;
	// URLPromptSentinel triggers the interactive URL prompt; any other
	// non-empty value is downloaded.
	URLFlag string

	// AllowStdin, when true, reads piped stdin as a fallback when no
	// explicit input mode was selected. Ignored when stdin is a TTY.
	AllowStdin bool

	// AutoDetectURL, when true, treats a positional argument that parses
	// as an http/https URL as a download source. Otherwise positional
	// args are always treated as file paths.
	AutoDetectURL bool

	// PickerTitle, PickerExts configure the interactive file picker used
	// when FileFlag == FilePickSentinel.
	PickerTitle string
	PickerExts  []string

	// URLPromptTitle, URLPromptPlaceholder configure the interactive URL
	// prompt used when URLFlag == URLPromptSentinel.
	URLPromptTitle       string
	URLPromptPlaceholder string

	// MaxBytes caps in-memory reads from stdin and URLs. 0 = 64 MB.
	// File paths are streamed straight from disk and are not affected.
	MaxBytes int64

	// HTTPTimeout is forwarded to the URL downloader. 0 = use the shared
	// httpclient timeout (set during startup).
	HTTPTimeout time.Duration
}

// defaultMaxBytes is applied to stdin/URL reads when Options.MaxBytes is 0.
const defaultMaxBytes = 64 << 20 // 64 MB

// ErrNoInput signals that no input method matched. Callers typically show
// command help in response. It is never wrapped, so callers can use
// errors.Is for a direct equality check.
var ErrNoInput = errors.New("no input provided")

// ErrNoTTY is returned when an interactive picker or prompt is requested
// but stdin is not a terminal. Callers can errors.Is against it to offer
// a non-interactive fallback or a clearer hint to the user.
var ErrNoTTY = errors.New("interactive prompt requires a terminal (stdin is piped or redirected)")

// Resolve reads the caller's selected input and returns its bytes along
// with a Source description. Priority (highest first):
//
//  1. FileFlag (sentinel triggers picker, else filesystem path)
//  2. URLFlag (sentinel triggers prompt, else URL)
//  3. PositionalArg ("-" means stdin; http/https URL when AutoDetectURL; else path)
//  4. Stdin pipe (only when AllowStdin is true and stdin is not a TTY)
//
// Returns ErrNoInput when none of the above matched. The context is
// honoured for URL downloads and stdin reads.
func Resolve(ctx context.Context, opts Options) ([]byte, Source, error) {
	path, src, err := resolvePath(opts)
	if err != nil {
		return nil, Source{}, err
	}

	// Path-driven branch: read the file from disk.
	if path != "" {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, Source{}, fmt.Errorf("reading %s: %w", path, readErr)
		}
		src.Size = int64(len(data))
		return data, src, nil
	}

	// URL branch: Resolve already populated src; fetch bytes.
	if src.Type == SourceURL {
		data, dErr := downloadURL(ctx, src.Path, opts)
		if dErr != nil {
			return nil, Source{}, dErr
		}
		src.Size = int64(len(data))
		return data, src, nil
	}

	// Stdin branch.
	if src.Type == SourceStdin {
		max := opts.MaxBytes
		if max <= 0 {
			max = defaultMaxBytes
		}
		data, rErr := io.ReadAll(io.LimitReader(os.Stdin, max+1))
		if rErr != nil {
			return nil, Source{}, fmt.Errorf("reading stdin: %w", rErr)
		}
		if int64(len(data)) > max {
			return nil, Source{}, fmt.Errorf("stdin exceeded %d byte cap", max)
		}
		src.Size = int64(len(data))
		return data, src, nil
	}

	return nil, Source{}, ErrNoInput
}

// ResolveStream returns a streaming reader for the selected input. Used
// by the hash command to avoid buffering large files in memory. URL
// downloads still buffer into memory (capped by MaxBytes) because the
// shared HTTP client does not currently expose a streaming body helper.
func ResolveStream(ctx context.Context, opts Options) (io.ReadCloser, Source, error) {
	path, src, err := resolvePath(opts)
	if err != nil {
		return nil, Source{}, err
	}

	if path != "" {
		f, oErr := os.Open(path)
		if oErr != nil {
			return nil, Source{}, fmt.Errorf("opening %s: %w", path, oErr)
		}
		if info, statErr := f.Stat(); statErr == nil {
			src.Size = info.Size()
		}
		return f, src, nil
	}

	if src.Type == SourceURL {
		data, dErr := downloadURL(ctx, src.Path, opts)
		if dErr != nil {
			return nil, Source{}, dErr
		}
		src.Size = int64(len(data))
		return io.NopCloser(strings.NewReader(string(data))), src, nil
	}

	if src.Type == SourceStdin {
		max := opts.MaxBytes
		if max <= 0 {
			max = defaultMaxBytes
		}
		// LimitReader + +1 lets callers detect an overflow after the fact.
		return io.NopCloser(io.LimitReader(os.Stdin, max+1)), src, nil
	}

	return nil, Source{}, ErrNoInput
}

// resolvePath dispatches input modes and returns (path, source, err).
// path is non-empty only for file-based inputs. For URL and stdin, source
// carries the identity and path is "".
func resolvePath(opts Options) (string, Source, error) {
	// --file (explicit path or picker)
	switch opts.FileFlag {
	case "":
		// unset; fall through
	case FilePickSentinel:
		// --file with no value after it -> picker, unless a positional
		// was also supplied (honours verify's historical behaviour).
		if opts.PositionalArg != "" && opts.PositionalArg != "-" {
			return opts.PositionalArg, Source{Type: SourceFile, Path: opts.PositionalArg}, nil
		}
		picked, err := pickFile(opts)
		if err != nil {
			return "", Source{}, err
		}
		return picked, Source{Type: SourcePicker, Path: picked}, nil
	default:
		return opts.FileFlag, Source{Type: SourceFile, Path: opts.FileFlag}, nil
	}

	// --url (explicit URL or prompt)
	switch opts.URLFlag {
	case "":
		// unset; fall through
	case URLPromptSentinel:
		if opts.PositionalArg != "" && isHTTPURL(opts.PositionalArg) {
			return "", Source{Type: SourceURL, Path: opts.PositionalArg}, nil
		}
		rawURL, err := promptURL(opts)
		if err != nil {
			return "", Source{}, err
		}
		return "", Source{Type: SourceURL, Path: rawURL}, nil
	default:
		return "", Source{Type: SourceURL, Path: opts.URLFlag}, nil
	}

	// Positional argument.
	if opts.PositionalArg != "" {
		if opts.PositionalArg == "-" {
			if !opts.AllowStdin {
				return "", Source{}, fmt.Errorf("'-' stdin alias not supported here")
			}
			return "", Source{Type: SourceStdin, Path: "-"}, nil
		}
		if opts.AutoDetectURL && isHTTPURL(opts.PositionalArg) {
			return "", Source{Type: SourceURL, Path: opts.PositionalArg}, nil
		}
		return opts.PositionalArg, Source{Type: SourceFile, Path: opts.PositionalArg}, nil
	}

	// Stdin pipe fallback.
	if opts.AllowStdin && IsStdinPipe() {
		return "", Source{Type: SourceStdin, Path: "-"}, nil
	}

	return "", Source{}, ErrNoInput
}

// IsStdinPipe returns true when stdin is connected to a pipe or file (not
// a terminal). Exported so commands that need to branch on the presence
// of piped input before calling Resolve can do so without importing os.
func IsStdinPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// IsStdinTerminal is the inverse of IsStdinPipe. Exported for clarity at
// call sites that gate interactive prompts on a real TTY.
func IsStdinTerminal() bool { return !IsStdinPipe() }

// isHTTPURL returns true for well-formed http(s) URLs with a host. The
// positional-arg path uses this to decide between a file and a URL.
func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// pickFileFunc and promptURLFunc are indirection hooks so tests can
// substitute a deterministic implementation for the interactive
// picker/prompt. In production they point at the huh-backed helpers
// defined below.
var (
	pickFileFunc  = defaultPickFile
	promptURLFunc = defaultPromptURL
)

// pickFile is the internal caller used by resolvePath; the indirection
// through pickFileFunc makes the interactive path testable.
func pickFile(opts Options) (string, error) { return pickFileFunc(opts) }

// promptURL is the internal caller used by resolvePath; see pickFile.
func promptURL(opts Options) (string, error) { return promptURLFunc(opts) }

// defaultPickFile launches the shared interactive picker. Fails fast with
// [ErrNoTTY] when stdin isn't a terminal — otherwise huh would crash deep
// in its render path with a cryptic error.
func defaultPickFile(opts Options) (string, error) {
	if !IsStdinTerminal() {
		return "", fmt.Errorf("--file without a path: %w", ErrNoTTY)
	}
	title := opts.PickerTitle
	if title == "" {
		title = "Select file"
	}
	return ui.PickFile(ui.PickFileOptions{Title: title, AllowedTypes: opts.PickerExts})
}

// defaultPromptURL collects a URL via a huh text input form. Fails fast
// with [ErrNoTTY] when stdin isn't a terminal — otherwise huh would
// crash deep in its render path with a cryptic error.
func defaultPromptURL(opts Options) (string, error) {
	if !IsStdinTerminal() {
		return "", fmt.Errorf("--url without a URL: %w", ErrNoTTY)
	}
	title := opts.URLPromptTitle
	if title == "" {
		title = "Enter URL"
	}
	placeholder := opts.URLPromptPlaceholder
	if placeholder == "" {
		placeholder = "https://example.com/..."
	}
	var rawURL string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder(placeholder).
				Value(&rawURL).
				Validate(validateURL),
		),
	).WithTheme(ui.HuhTheme()).Run()
	if err != nil {
		return "", fmt.Errorf("URL input: %w", err)
	}
	if rawURL == "" {
		return "", fmt.Errorf("no URL provided")
	}
	return rawURL, nil
}

// validateURL is the tiny pure-function predicate behind the huh
// input's Validate() callback. Extracted so tests can exercise it
// without running the terminal form.
func validateURL(s string) error {
	if s == "" {
		return nil
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return fmt.Errorf("must start with http:// or https://")
	}
	return nil
}

// downloadURL validates the URL shape and fetches bytes through the
// shared HTTP client. Kept generic — does not assume a JSON body.
func downloadURL(ctx context.Context, rawURL string, opts Options) ([]byte, error) {
	if !isHTTPURL(rawURL) {
		return nil, fmt.Errorf("URL must be http:// or https:// with a host: %q", rawURL)
	}
	max := opts.MaxBytes
	if max <= 0 {
		max = defaultMaxBytes
	}
	data, err := httpclient.DownloadBytesCtx(ctx, rawURL, max)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", rawURL, err)
	}
	return data, nil
}
