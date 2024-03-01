package utils

import (
	"encoding/json"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/olekukonko/tablewriter"
)

type OutputFormat interface {
	Caption() string
	JSONFormat() string
	TableFormat() []string
	TableHeaders() []string
}

// BuildTable builds the table from the table header and structured
// table data and returns a string that can be outputted to the command
// or stdout.
func BuildTable(of []OutputFormat) (string, error) {
	tableOutput := &strings.Builder{}
	table := tablewriter.NewWriter(tableOutput)

	headers := []string{}
	caption := ""

	if len(of) > 0 {
		headers = of[0].TableHeaders()
		caption = of[0].Caption()
	}
	table.SetHeader(headers)
	table.SetCaption(true, caption)
	for _, item := range of {
		if len(item.TableHeaders()) != len(item.TableFormat()) {
			return "", errors.Newf("number of column headers %s doesn't match the number of data fields %s", item.TableHeaders(), item.TableFormat())
		}

		table.Append(item.TableFormat())
	}
	table.Render()

	return tableOutput.String(), nil
}

func PrettyJSON(v any) string {
	data, _ := json.MarshalIndent(v, "", "    ")
	return string(data)
}
