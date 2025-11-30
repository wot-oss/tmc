package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var TmcVersion = "n/a"

func GetTmcVersion() string {
	v, err := semver.NewVersion(TmcVersion)
	if err != nil {
		return TmcVersion
	}
	return strings.TrimPrefix(v.Original(), "v")
}

// ReadRequiredFile reads the file. Returns expanded absolute representation of the filename and file contents.
// Removes Byte-Order-Mark from the content
func ReadRequiredFile(name string) (string, []byte, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", nil, fmt.Errorf("error expanding file name %s: %w", name, err)
	}

	stat, err := os.Stat(abs)
	if err != nil {
		return "", nil, fmt.Errorf("error reading file %s: %w", abs, err)
	}
	if stat.IsDir() {
		return "", nil, fmt.Errorf("%s is not a file", abs)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return "", nil, fmt.Errorf("error reading file %s: %w", abs, err)
	}
	raw = removeBOM(raw)
	return abs, raw, nil
}

func removeBOM(bytes []byte) []byte {
	if len(bytes) > 2 && bytes[0] == 0xef && bytes[1] == 0xbb && bytes[2] == 0xbf {
		bytes = bytes[3:]
	}
	return bytes
}

// ExpandHome expands ~ in path with user's home directory, but only if path begins with ~ or /~
// Otherwise, returns path unchanged
func ExpandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") && !strings.HasPrefix(path, "/~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot expand user home directory: %w", err)
	}
	_, rest, found := strings.Cut(path, "~")
	if !found {
		panic(errors.New("should have checked for ~ before"))
	}
	return filepath.Join(home, rest), nil
}

func ToTrimmedLower(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return s
}

func NormalizeLineEndings(bytes []byte) []byte {
	res := make([]byte, 0, len(bytes))
	var prevB byte
	for _, b := range bytes {
		switch b {
		case '\n':
			if prevB != '\r' {
				res = append(res, '\n')
			}
		case '\r':
			res = append(res, '\n')
		default:
			res = append(res, b)
		}
		prevB = b
	}
	return res
}

func JsGetBool(js map[string]any, key string) (val bool, found bool) {
	if v, ok := js[key]; ok {
		if b, ok := v.(bool); ok {
			return b, true
		}
	}
	return false, false
}
func JsGetString(js map[string]any, key string) (val string, found bool) {
	if v, ok := js[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

func JsGetMap(js map[string]any, key string) (val map[string]any, found bool) {
	if v, ok := js[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m, true
		}
	}
	return nil, false
}
func JsGetArray(js map[string]any, key string) []any {
	if v, ok := js[key]; ok {
		if m, ok := v.([]any); ok {
			return m
		}
	}
	return nil
}

// ConvertToNativeLineEndings converts all instances of '\n' to native line endings for the platform.
// Assumes that line endings are normalized, i.e. there are no '\r' or "\r\n" line endings in the data
// See NormalizeLineEndings
func ConvertToNativeLineEndings(b []byte) []byte {
	return convertToNativeLineEndings(b)
}

func EncodeJSONWithoutEscapeHTML(v any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("unexpected encoding error %w", err)
	}
	return buffer.Bytes(), nil
}

// AtomicWriteFile writes data to the named file quasi-atomically, creating it if necessary.
// On unix-like systems, the function uses github.com/google/renameio.
// On Windows, it has a simpler implementation using os.Rename(), which is believed to be atomic on NTFS,
// but there is no hard guarantee from Microsoft on that.
func AtomicWriteFile(name string, data []byte, perm os.FileMode) error {
	return atomicWriteFile(name, data, perm)
}

func ParseAsList(list, separator string, trim bool) []string {
	ret := make([]string, 0)

	for _, entry := range strings.Split(list, separator) {
		if trim {
			entry = strings.TrimSpace(entry)
		}
		if entry != "" {
			ret = append(ret, entry)
		}
	}
	return ret
}

// ReadFileLines reads a whole file into memory and returns its lines.
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// WriteFileLines writes the lines to the given file.
func WriteFileLines(lines []string, path string, mode os.FileMode) error {
	buf := bytes.NewBuffer(nil)
	for _, line := range lines {
		_, err := fmt.Fprintln(buf, line)
		if err != nil {
			return err
		}
	}
	return AtomicWriteFile(path, buf.Bytes(), mode)
}

var (
	removableChars   = regexp.MustCompile(`[^\[a-zA-Z0-9-]`)
	replaceableChars = regexp.MustCompile(`[ &_=+:/]`)
	dashes           = regexp.MustCompile(`[\-]+`)

	accents = map[rune]string{
		'à': "a",
		'á': "a",
		'â': "a",
		'ã': "a",
		'ä': "ae",
		'å': "aa",
		'æ': "ae",
		'ç': "c",
		'è': "e",
		'é': "e",
		'ê': "e",
		'ë': "e",
		'ì': "i",
		'í': "i",
		'î': "i",
		'ï': "i",
		'ð': "d",
		'ł': "l",
		'ñ': "n",
		'ń': "n",
		'ò': "o",
		'ó': "o",
		'ô': "o",
		'õ': "o",
		'ō': "o",
		'ö': "oe",
		'ø': "oe",
		'œ': "oe",
		'ś': "s",
		'ù': "u",
		'ú': "u",
		'û': "u",
		'ū': "u",
		'ü': "ue",
		'ý': "y",
		'ÿ': "y",
		'ż': "z",
		'þ': "th",
		'ß': "ss",
	}
)

func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return name
	}
	name = strings.ToLower(name)
	name = replaceableChars.ReplaceAllString(name, "-")
	name = sanitizeAccents(name)
	name = removableChars.ReplaceAllString(name, "")
	name = dashes.ReplaceAllString(name, "-")
	return name
}

func sanitizeAccents(s string) string {
	bs := bytes.NewBufferString("")
	for _, c := range s {
		if val, ok := accents[c]; ok {
			bs.WriteString(val)
		} else {
			bs.WriteRune(c)
		}
	}
	return bs.String()
}

type ReadCloserGetter func() (io.ReadCloser, error)

func ReadCloserGetterFromBytes(raw []byte) ReadCloserGetter {
	return func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(raw)), nil }
}

func ReadCloserGetterFromFilename(name string) ReadCloserGetter {
	return func() (io.ReadCloser, error) { return os.Open(name) }
}

// DetectMediaType detects the media type of the file. The type provided by the user always takes precedence over
// automatic detection, unless it is empty. The type is detected by http.DetectContentType. If that returns the
// generic 'application/octet-stream', then the type is guessed from the filename extension.
// If all of the above fails, it returns 'application/octet-stream'
func DetectMediaType(userGivenType string, filename string, getReader ReadCloserGetter) string {
	const mediaOctetStream = "application/octet-stream"
	if userGivenType != "" {
		return userGivenType
	}

	reader, err := getReader()
	if err == nil {
		defer reader.Close()
		truncatedContent, err := io.ReadAll(io.LimitReader(reader, 512))
		if err == nil {
			ct := http.DetectContentType(truncatedContent)
			if ct != mediaOctetStream {
				return ct
			}
		}
	}

	ct := mime.TypeByExtension(filepath.Ext(filename))
	if ct != "" {
		return ct
	}
	return mediaOctetStream
}

const CtxKeyLogger = "logger"

// GetLogger returns the logger that is valid in the context
// If component is not empty, the logger is extended with the field "where" having that value.
func GetLogger(ctx context.Context, component string) *slog.Logger {
	cv := ctx.Value(CtxKeyLogger)
	l, ok := cv.(*slog.Logger)
	if !ok || l == nil {
		l = slog.Default()
	}
	if component != "" {
		l = l.With("where", component)
	}
	return l
}
