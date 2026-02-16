package date

import (
	"embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"date", "date -u", "date +%F", "date +%s"}

//go:embed manual.md
var manualFS embed.FS

func detailedManual() string {
	data, err := manualFS.ReadFile("manual.md")
	if err != nil {
		return ""
	}
	return string(data)
}

func Spec() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   "date",
		Manual: "date [-u] [+%Y-%m-%d|+%F|+%T|+%s|...]",
		Tips: []string{
			"Use -u for UTC time output.",
			"Format specifiers follow the supported simsh subset only.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Capabilities:   []string{"compat:date"},
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	_ = runtime
	useUTC := false
	formatSpec := ""
	for _, arg := range args {
		switch {
		case arg == "-u":
			useUTC = true
		case strings.HasPrefix(arg, "+"):
			if formatSpec != "" {
				return fmt.Sprintf("date: unexpected argument: %s", arg), contract.ExitCodeUsage
			}
			formatSpec = strings.TrimPrefix(arg, "+")
		default:
			return fmt.Sprintf("date: unexpected argument: %s", arg), contract.ExitCodeUsage
		}
	}

	now := time.Now()
	if useUTC {
		now = now.UTC()
	}
	if formatSpec == "" {
		return now.Format("2006-01-02 15:04:05 -0700 MST") + "\n", 0
	}
	formatted, err := formatDateSpec(now, formatSpec)
	if err != nil {
		return fmt.Sprintf("date: %v", err), contract.ExitCodeUsage
	}
	return formatted + "\n", 0
}

func formatDateSpec(now time.Time, formatSpec string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(formatSpec); i++ {
		ch := formatSpec[i]
		if ch != '%' {
			out.WriteByte(ch)
			continue
		}
		i++
		if i >= len(formatSpec) {
			return "", fmt.Errorf("invalid format: trailing %%")
		}
		switch formatSpec[i] {
		case '%':
			out.WriteByte('%')
		case 'Y':
			out.WriteString(now.Format("2006"))
		case 'm':
			out.WriteString(now.Format("01"))
		case 'd':
			out.WriteString(now.Format("02"))
		case 'H':
			out.WriteString(now.Format("15"))
		case 'M':
			out.WriteString(now.Format("04"))
		case 'S':
			out.WriteString(now.Format("05"))
		case 'F':
			out.WriteString(now.Format("2006-01-02"))
		case 'T':
			out.WriteString(now.Format("15:04:05"))
		case 'z':
			out.WriteString(now.Format("-0700"))
		case 'Z':
			out.WriteString(now.Format("MST"))
		case 's':
			out.WriteString(strconv.FormatInt(now.Unix(), 10))
		default:
			return "", fmt.Errorf("unsupported format %%%c", formatSpec[i])
		}
	}
	return out.String(), nil
}
