package testutils

import "github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"

type ExportMode int

const (
	ExportWithSelect ExportMode = iota
	ExportWithCopy
)

type FetchTestingKnobs struct {
	// Used to simulate testing when the CSV input file is wrong.
	TriggerCorruptCSVFile bool

	FailedWriteToBucket FailedWriteToBucketKnob

	FailedEstablishSrcConnForExport bool

	// To simulate failure when exporting a certain shard.
	FailedToExportForShard *FailedToExportForShardKnob

	ExpMode ExportMode
}

type FailedWriteToBucketKnob struct {
	FailedBeforeReadFromPipe bool
	FailedAfterReadFromPipe  bool
}

type FailedToExportForShardKnob struct {
	FailedExportDataToPipeCondition                 func(tableName tree.Name, shardIdx int) bool
	FailedReadDataFromPipeInitWriterCondition       func(tableName tree.Name, shardIdx int, itNum int) bool
	FailedReadDataFromPipeWriteToCSVWriterCondition func(tableName tree.Name, shardIdx int, rowCnt int) bool
}
