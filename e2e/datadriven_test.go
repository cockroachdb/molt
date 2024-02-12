package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cockroachdb/datadriven"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/testutils"
	"github.com/stretchr/testify/require"
)

func TestDataDriven(t *testing.T) {
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		driverDialect := filepath.Base(filepath.Dir(path))

		require.NoError(t, Setup(driverDialect))
		t.Logf("finished setup")
		require.NoError(t, ConfirmContainersRunning(t, driverDialect))
		t.Logf("containers are all up")

		defer func() {
			t.Logf("tearing down containers")
			require.NoError(t, TearDown())
			t.Logf("all containers are terminated")
		}()

		datadriven.RunTestAny(t, path, func(t testing.TB, d *datadriven.TestData) string {
			// Remove common args.
			var silent bool
			newArgs := d.CmdArgs[:0]

			for _, arg := range d.CmdArgs {
				switch arg.Key {
				case "silent":
					silent = true
					continue
				}
				newArgs = append(newArgs, arg)
			}
			d.CmdArgs = newArgs

			switch d.Cmd {
			case "exec":
				var stdout strings.Builder
				var stderr strings.Builder
				cmd := exec.Command("/bin/sh", "-c", d.Input)
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				err := cmd.Run()
				if err != nil {
					t.Fatalf(errors.Wrapf(errors.New(stderr.String()), "error executing %s", strings.Join(cmd.Args, " ")).Error())
				}
				if silent {
					return ""
				}
				return strings.TrimSpace(stdout.String())
			case "fetch", "verify":
				if len(d.CmdArgs) < 3 {
					t.Fatalf("expect at least 2 args for %q command", d.Cmd)
				}

				toRunCmd := fmt.Sprintf(`go run .. %s --test-only %s`, d.Cmd, testutils.GetCmdArgsStr(d.CmdArgs))
				t.Logf("running %q", toRunCmd)
				cmd := exec.Command("/bin/sh", "-c", toRunCmd)

				var stdout strings.Builder
				var stderr strings.Builder
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				err := cmd.Run()
				if err != nil {
					t.Fatalf(errors.Wrapf(errors.New(stderr.String()), "error executing molt %s", d.Cmd).Error())
				}
				return strings.TrimSpace(redactLogs(stdout.String()))
			}
			t.Fatalf("unknown command: %s", d.Cmd)
			return ""
		})
	})
}

// redactLogs is to remove fields that are not deterministic.
func redactLogs(s string) string {
	// Remove the "time:xxxxx" filed of the Info logs.
	const timePattern = `\"time\":\"[0-9T\-\:TZ]*\",`
	res := regexp.MustCompile(timePattern).ReplaceAllString(s, "")

	// Remove the `starting file server` log as it is logged from a goroutine
	// whose occurrence order is not deterministic with the main log flow.

	// (?m) is added at the beginning of the pattern to enable multi-line mode.
	// This allows ^ (and $) to match the start and end of each line, not just
	// the entire string.
	const fileServerStartingPattern = `(?m)^[^\n]*starting file server[^\n]*\n?`
	return regexp.MustCompile(fileServerStartingPattern).ReplaceAllString(res, "")
}
