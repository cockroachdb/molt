package typeconv

import "fmt"

type ShortDesc = string

const (
	InvalidDecimalArgs       ShortDesc = "Invalid decimal args"
	UnsupportedBytesMax      ShortDesc = "Bytes limit not supported"
	UnsupportedColumnTypeRaw ShortDesc = "Unsupported column type"
	UnsupportedTinyInt       ShortDesc = "TINYINT not supported"
)

var (
	UnsupportedCollate    = func(c string) ShortDesc { return fmt.Sprintf("collate %s not supported", c) }
	UnsupportedColumnType = func(c string) ShortDesc { return fmt.Sprintf("%s %s", UnsupportedColumnTypeRaw, c) }
)
