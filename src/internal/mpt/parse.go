package mpt

import (
	"regexp"
	"strings"
)

func clean(s string) string {
	s = strings.TrimRight(s, ".,")
	return s
}

// HistoryLatestFromOutput parses `monad-mpt` stdout and returns history latest block (same as get_mpt_history_latest).
func HistoryLatestFromOutput(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "MPT database has") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 11 {
			return clean(fields[10])
		}
	}
	return ""
}

// Info holds parsed fields from get_mpt_info awk block.
type Info struct {
	Path              string
	Capacity          string
	Used              string
	UsedPercent       string
	HistoryCount      string
	HistoryEarliest   string
	HistoryLatest     string
	HistoryRetention  string
	LatestFinalized   string
	LatestVerified    string
	AutoExpireVersion string
}

var firstLineRE = regexp.MustCompile(`^\s*([0-9.]+\s+[A-Za-z]+)\s+([0-9.]+\s+[A-Za-z]+)\s+([0-9.]+)%\s+"([^"]+)"`)

// ParseInfo parses monad-mpt output for triedb summary (best-effort).
func ParseInfo(output string) *Info {
	info := &Info{}
	for _, line := range strings.Split(output, "\n") {
		if m := firstLineRE.FindStringSubmatch(line); len(m) == 5 {
			info.Capacity = strings.TrimSpace(m[1])
			info.Used = strings.TrimSpace(m[2])
			info.UsedPercent = m[3]
			info.Path = m[4]
		}
		if strings.Contains(line, "MPT database has") {
			f := strings.Fields(line)
			if len(f) >= 11 {
				info.HistoryCount = clean(f[3])
				info.HistoryEarliest = clean(f[7])
				info.HistoryLatest = clean(f[10])
			}
		}
		if strings.Contains(line, "retain no more than") {
			f := strings.Fields(line)
			if len(f) > 0 {
				info.HistoryRetention = clean(f[len(f)-1])
			}
		}
		if strings.Contains(line, "Latest finalized is") {
			f := strings.Fields(line)
			if len(f) >= 8 {
				info.LatestFinalized = clean(f[3])
				info.LatestVerified = clean(f[7])
			}
			if len(f) > 0 {
				info.AutoExpireVersion = clean(f[len(f)-1])
			}
		}
	}
	return info
}
