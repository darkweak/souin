package templates

import (
	"errors"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"go.step.sm/crypto/jose"
)

// GetFuncMap returns the list of functions provided by sprig. It adds the
// functions "toTime", "formatTime", "parseTime", "mustParseTime",
// "toTimeLayout" and changes the function "fail".
//
// The "toTime" function receives a time or a Unix epoch and returns a time.Time
// in UTC. The "formatTime" function uses "toTime" and formats the resulting
// time using RFC3339. The functions "parseTime" and "mustParseTime" parse a
// string and return the time.Time it represents. The "toTimeLayout" function
// converts strings like "time.RFC3339" or "UnixDate" to the actual layout
// represented by the Go constant with the same name. The "fail" function sets
// the provided message, so that template errors are reported directly to the
// template without having the wrapper that text/template adds.
//
//	{{ toTime }}
//	    => time.Now().UTC()
//	{{ .Token.nbf | toTime }}
//	    => time.Unix(.Token.nbf, 0).UTC()
//	{{ .Token.nbf | formatTime }}
//	    => time.Unix(.Token.nbf, 0).UTC().Format(time.RFC3339)
//	{{ "2024-07-02T23:16:02Z" | parseTime }}
//	    => time.Parse(time.RFC3339, "2024-07-02T23:16:02Z")
//	{{ parseTime "time.RFC339" "2024-07-02T23:16:02Z" }}
//	    => time.Parse(time.RFC3339, "2024-07-02T23:16:02Z")
//	{{ parseTime "time.UnixDate" "Tue Jul  2 16:20:48 PDT 2024" "America/Los_Angeles" }}
//	    => loc, _ := time.LoadLocation("America/Los_Angeles")
//	       time.ParseInLocation(time.UnixDate, "Tue Jul  2 16:20:48 PDT 2024", loc)
//	{{ toTimeLayout "RFC3339" }}
//	    => time.RFC3339
//
// sprig "env" and "expandenv" functions are removed to avoid the leak of
// information.
func GetFuncMap(failMessage *string) template.FuncMap {
	m := sprig.TxtFuncMap()
	delete(m, "env")
	delete(m, "expandenv")
	m["fail"] = func(msg string) (string, error) {
		*failMessage = msg
		return "", errors.New(msg)
	}
	m["formatTime"] = formatTime
	m["toTime"] = toTime
	m["parseTime"] = parseTime
	m["mustParseTime"] = mustParseTime
	m["toTimeLayout"] = toTimeLayout
	return m
}

func toTime(v any) time.Time {
	var t time.Time
	switch date := v.(type) {
	case time.Time:
		t = date
	case *time.Time:
		t = *date
	case int64:
		t = time.Unix(date, 0)
	case float64: // from json
		t = time.Unix(int64(date), 0)
	case int:
		t = time.Unix(int64(date), 0)
	case int32:
		t = time.Unix(int64(date), 0)
	case jose.NumericDate:
		t = date.Time()
	case *jose.NumericDate:
		t = date.Time()
	default:
		t = time.Now()
	}
	return t.UTC()
}

func formatTime(v any) string {
	return toTime(v).Format(time.RFC3339)
}

func parseTime(v ...string) time.Time {
	t, _ := mustParseTime(v...)
	return t
}

func mustParseTime(v ...string) (time.Time, error) {
	switch len(v) {
	case 0:
		return time.Now().UTC(), nil
	case 1:
		return time.Parse(time.RFC3339, v[0])
	case 2:
		layout := toTimeLayout(v[0])
		return time.Parse(layout, v[1])
	case 3:
		layout := toTimeLayout(v[0])
		loc, err := time.LoadLocation(v[2])
		if err != nil {
			return time.Time{}, err
		}
		return time.ParseInLocation(layout, v[1], loc)
	default:
		return time.Time{}, errors.New("unsupported number of parameters")
	}
}

func toTimeLayout(fmt string) string {
	switch strings.ToUpper(strings.TrimPrefix(fmt, "time.")) {
	case "LAYOUT":
		return time.Layout
	case "ANSIC":
		return time.ANSIC
	case "UNIXDATE":
		return time.UnixDate
	case "RUBYDATE":
		return time.RubyDate
	case "RFC822":
		return time.RFC822
	case "RFC822Z":
		return time.RFC822Z
	case "RFC850":
		return time.RFC850
	case "RFC1123":
		return time.RFC1123
	case "RFC1123Z":
		return time.RFC1123Z
	case "RFC3339":
		return time.RFC3339
	case "RFC3339NANO":
		return time.RFC3339Nano
	// From the ones below, only time.DateTime will parse a complete date.
	case "KITCHEN":
		return time.Kitchen
	case "STAMP":
		return time.Stamp
	case "STAMPMILLI":
		return time.StampMilli
	case "STAMPMICRO":
		return time.StampMicro
	case "STAMPNANO":
		return time.StampNano
	case "DATETIME":
		return time.DateTime
	case "DATEONLY":
		return time.DateOnly
	case "TIMEONLY":
		return time.TimeOnly
	default:
		return fmt
	}
}
