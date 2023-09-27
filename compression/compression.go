package compression

import "github.com/thediveo/enumflag/v2"

//go:generate go run github.com/alvaroloes/enumer -type=Flag -output compression_enumer.gen.go
type Flag enumflag.Flag

const (
	Default Flag = iota + 1
	GZIP
	None
)

var CompressionStringRepresentations = map[Flag][]string{
	Default: {"default"},
	GZIP:    {"gzip"},
	None:    {"none"},
}
