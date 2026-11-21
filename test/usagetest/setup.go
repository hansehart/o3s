package usagetest

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
)

// table is the setups PICT generated from testdata/setups.pict.
//
//go:embed testdata/setups.golden
var table string

// Setup is one setup the usage suite runs o3s on.
type Setup struct {
	// Session is where the user's terminal is: "local" on the machine, or "ssh" on it over SSH.
	Session string
	// Agent is the SSH agent the session holds: "none", or "host" for one with a key of its own.
	Agent string
	// Cols and Rows are the size of the user's terminal.
	Cols, Rows int
	// Fault is what breaks halfway through, "none" where nothing does.
	Fault string
}

// Name names a setup the way a test's name and a failure name it.
func (s Setup) Name() string {
	return fmt.Sprintf("%s-%s-%dx%d-%s", s.Session, s.Agent, s.Cols, s.Rows, s.Fault)
}

// Setups are the setups of the table, in its order.
func Setups() ([]Setup, error) {
	lines := strings.Split(strings.TrimSpace(table), "\n")
	// Turn away a table whose columns are other than the model's
	if lines[0] != "Session\tAgent\tSize\tFault" {
		return nil, fmt.Errorf("setups.golden columns %q, want Session, Agent, Size and Fault", lines[0])
	}
	out := make([]Setup, 0, len(lines)-1)
	// Read every row below the columns
	for _, line := range lines[1:] {
		f := strings.Split(strings.TrimSpace(line), "\t")
		// Turn away a row of another width
		if len(f) != 4 {
			return nil, fmt.Errorf("setups.golden row %q, want 4 columns", line)
		}
		c, r, ok := strings.Cut(f[2], "x")
		cols, cerr := strconv.Atoi(c)
		rows, rerr := strconv.Atoi(r)
		// Turn away a size that reads as none
		if !ok || cerr != nil || rerr != nil {
			return nil, fmt.Errorf("setups.golden size %q, want COLSxROWS", f[2])
		}
		out = append(out, Setup{Session: f[0], Agent: f[1], Cols: cols, Rows: rows, Fault: strings.TrimPrefix(f[3], "~")})
	}
	return out, nil
}
