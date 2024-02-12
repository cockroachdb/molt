package testutils

import (
	"strings"

	"github.com/cockroachdb/datadriven"
)

func GetCmdArgsStr(in []datadriven.CmdArg) string {
	var cmdArgStrs []string
	for _, arg := range in {
		cmdArgStrs = append(cmdArgStrs, arg.String())
	}
	return strings.Join(cmdArgStrs, " ")
}
