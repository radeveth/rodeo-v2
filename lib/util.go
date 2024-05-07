package lib

import (
	crand "crypto/rand"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jmoiron/sqlx"
	"gitlab.com/golang-commonmark/markdown"
	"golang.org/x/text/unicode/norm"
)

func init() {
	sqlx.NameMapper = StringToSnakeCase
}

// SessionCookieName is the name of the session cookie to use
var SessionCookieName = "sessionid"
var SessionSigninRedirect = "/"

// J is a shorthand for map[string]interface{}. Often used to represent JSON or pass map data around.
type J map[string]interface{}

func (j J) Set(key string, value interface{}) {
	j[key] = value
}
func (j J) Get(key string) string {
	return IToString(j[key])
}
func (j J) GetInt(key string) int64 {
	return IToInt(j[key])
}
func (j J) GetBool(key string) bool {
	return IToBool(j[key])
}
func (j J) GetTime(key string) time.Time {
	return j[key].(time.Time)
}
func (j J) GetJ(key string) J {
	if v, ok := j[key].(map[string]interface{}); ok {
		return J(v)
	}
	return j[key].(J)
}

// Value serializes a J instance for the sql database driver
func (j J) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Scan deserializes a value to a J instance for the sql database driver
func (j *J) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("scan: type assertion to []byte failed")
	}
	return json.Unmarshal(b, j)
}

func ExecCmd(command string) string {
	Log("debug", "ExecCmd: Running command", J{"command": command})
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = os.Environ()
	bs, err := cmd.CombinedOutput()
	if err != nil {
		Log("error", "ExecCmd: Error running commmand", J{"command": command, "error": err.Error(), "output": string(bs)})
		os.Exit(1)
	}
	return string(bs)
}

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

// NewLoggedError logs the given error message and returns a new error for it
func NewLoggedError(format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	errorLogger.Println(err.Error())
	return err
}

// Log logs the given message and values out to standard out as JSON
func Log(level string, message string, extra ...J) {
	fields := J{}
	if len(extra) > 0 {
		fields = extra[0]
	}
	t := time.Now().UTC().Format(time.RFC3339)
	if Env("ENV", "") != "development" {
		bs, _ := json.Marshal(J{
			"time":    t,
			"level":   level,
			"message": message,
			"fields":  fields,
		})
		fmt.Println(string(bs))
	} else {
		color := ""
		if level == "error" {
			color = "\x1b[31m"
		} else if level == "warning" {
			color = "\x1b[33m"
		} else if level == "info" {
			color = "\x1b[34m"
		}
		fbs, _ := json.Marshal(fields)
		if len(fbs) > 120 {
			fbs, _ = json.MarshalIndent(fields, "", "  ")
		}
		t = time.Now().Format("2006-01-02 15:04")
		level = string(strings.ToUpper(level)[0])
		fmt.Printf("\x1b[1m%s\x1b[0m %s[%s] %s\x1b[0m %s\n", t, color, level, message, string(fbs))
	}
}

func LogDebug(message string, extra ...J) {
	Log("debug", message, extra...)
}

func LogInfo(message string, extra ...J) {
	Log("info", message, extra...)
}

func LogError(message string, extra ...J) {
	Log("error", message, extra...)
}

// Check panic if given a non-nil error.
// Useful to make sure to end the execution of a method when an error is encountered and `error` is not part of the return values
func Check(err error) {
	if err != nil {
		panic(err)
	}
}

// Env returns the environment variable value for `name` or the provided `alt` value it it's not set or empty.
func Env(name, alt string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return alt
}

// EnvBool returns true when the environment variable for `name` is equal to "1".
func EnvBool(name string) bool {
	return os.Getenv(name) == "1"
}

const idEncoding = "0123456789ABCDEFGHJKMNPQRSTVWXYZabcdefghkmnpqrstvwxyz"

// NewID returns a new ID where the first 10 characters represents a timestamp
// and last 10 characters are chosen at random.
func NewID() string {
	enclen := len(idEncoding)
	id := make([]byte, 20)
	for i := 0; i < 10; i++ {
		id[10+i] = idEncoding[rand.Intn(enclen)]
	}
	now := time.Now().UnixNano() / int64(time.Millisecond)
	for i := 0; i < 10; i++ {
		mod := now % int64(enclen)
		id[9-i] = idEncoding[mod]
		now = (now - mod) / int64(enclen)
	}
	return string(id)
}

// NewRandomID returns an new ID where all 20 characters are chosen at random
func NewRandomID() string {
	enclen := len(idEncoding)
	id := make([]byte, 20)
	for i := 0; i < 20; i++ {
		v, err := crand.Int(crand.Reader, big.NewInt(int64(enclen)))
		Check(err)
		id[i] = idEncoding[v.Int64()]
	}
	return string(id)
}

// Min for ints
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max for ints
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// IntToString converts an int to a string
func IntToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// StringToInt converts a string to an int
func StringToInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	Check(err)
	return i
}

func StringToFloat(s string) float64 {
	n, err := strconv.ParseFloat(s, 64)
	Check(err)
	return n
}

// StringToLocation converts a timezone name to a Location instance
func StringToLocation(timezone string) *time.Location {
	l, err := time.LoadLocation(timezone)
	Check(err)
	return l
}

// MarkdownToString converts markdown to an html string
func MarkdownToString(text string) string {
	md := markdown.New(
		markdown.HTML(true),
		markdown.Tables(true),
		markdown.Typographer(true),
		markdown.XHTMLOutput(true),
		markdown.Nofollow(true))
	return md.RenderToString([]byte(text))
}

// StringToTitle capitalizes the 1st letter of the provided string
func StringToTitle(word string) string {
	return strings.ToUpper(word[:1]) + word[1:]
}

// StringToCamelCase converts a snake cased string to it's camel case format
func StringToCamelCase(str string) (out string) {
	parts := strings.Split(strings.ToLower(str), "_")
	for i, part := range parts {
		if i == 0 {
			out += part
		} else {
			out += StringToTitle(part)
		}
	}
	return out
}

// StringToSnakeCase converts a camel cased string to it's snake case format
func StringToSnakeCase(src string) string {
	buf := ""
	for i, v := range src {
		if i > 0 && isUpper(v) && !isUpper([]rune(src)[i-1]) {
			buf += "_"
		}
		buf += string(v)
	}
	buf = strings.Replace(buf, "/", "_", -1)
	return strings.ToLower(buf)
}

var slugSkip = []*unicode.RangeTable{
	unicode.Mark,
	unicode.Sk,
	unicode.Lm,
}

var slugSafe = []*unicode.RangeTable{
	unicode.Letter,
	unicode.Number,
}

func StringToSlug(text string) string {
	buf := make([]rune, 0, len(text))
	dash := false
	for _, r := range norm.NFKD.String(text) {
		switch {
		case unicode.IsOneOf(slugSafe, r):
			buf = append(buf, unicode.ToLower(r))
			dash = true
		case unicode.IsOneOf(slugSkip, r):
		case dash:
			buf = append(buf, '-')
			dash = false
		}
	}
	if i := len(buf) - 1; i >= 0 && buf[i] == '-' {
		buf = buf[:i]
	}
	return string(buf)
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// IToString converts an interface to string
func IToString(i interface{}) string {
	if v, ok := i.(string); ok {
		return v
	}
	if v, ok := i.(int64); ok {
		return IntToString(v)
	}
	if v, ok := i.(float64); ok {
		return IntToString(int64(v))
	}
	if v, ok := i.(int); ok {
		return IntToString(int64(v))
	}
	return ""
}

// IToInt converts an interface to int
func IToInt(i interface{}) int64 {
	if v, ok := i.(int64); ok {
		return v
	}
	if v, ok := i.(float64); ok {
		return int64(v)
	}
	if v, ok := i.(int); ok {
		return int64(v)
	}
	return 0
}

// IToBool converts an interface to bool
func IToBool(i interface{}) bool {
	if v, ok := i.(bool); ok {
		return v
	}
	return false
}

// IToTime converts an interface to Time
func IToTime(i interface{}) time.Time {
	if v, ok := i.(time.Time); ok {
		return v
	}
	return time.Time{}
}

func IsProduction() bool {
	return Env("ENV", "development") != "development"
}
