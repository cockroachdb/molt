package typeconv

import "fmt"

type ShortDesc = string

const (
	CrossDatabaseQualifier             ShortDesc = "Cross-database qualifier not supported"
	DBOSchema                          ShortDesc = "DBO schema converted"
	InvalidDecimalArgs                 ShortDesc = "Invalid decimal args"
	InvalidModifyColumn                ShortDesc = "Invalid modify column"
	MissingComputedColumnType          ShortDesc = "Could not determine computed column type"
	Unparsable                         ShortDesc = "Does not parse"
	UnparsableExpr                     ShortDesc = "Expression does not parse"
	UnparsableTypeLiteral              ShortDesc = "Unparsable type literal"
	UnsupportedAuthorization           ShortDesc = "AUTHORIZATION not supported"
	UnsupportedBytesMax                ShortDesc = "Bytes limit not supported"
	UnsupportedClustered               ShortDesc = "Unsupported CLUSTERED option"
	UnsupportedColumnDef               ShortDesc = "Unsupported column definition"
	UnsupportedColumnEncryption        ShortDesc = "Column encryption not supported"
	UnsupportedColumnOption            ShortDesc = "Unsupported column option"
	UnsupportedColumnStore             ShortDesc = "COLUMNSTORE not supported"
	UnsupportedColumnTypeRaw           ShortDesc = "Unsupported column type"
	UnsupportedConstraint              ShortDesc = "Unsupported constraint type"
	UnsupportedConstraintState         ShortDesc = "Unsupported constraint state"
	UnsupportedDisabledConstraint      ShortDesc = "Unsupported disabled constraint"
	UnsupportedErrorLoggingClause      ShortDesc = "Unsupported error logging clause"
	UnsupportedFilestream              ShortDesc = "Unsupported FILESTREAM"
	UnsupportedIndexOption             ShortDesc = "Unsupported index option"
	UnsupportedMasked                  ShortDesc = "Unsupported MASKED clause"
	UnsupportedNonclustered            ShortDesc = "Unsupported NONCLUSTERED clause"
	UnsupportedNotForReplication       ShortDesc = "NOT FOR REPLICATION not supported"
	UnsupportedOrganizationClause      ShortDesc = "Unsupported ORGANIZATION"
	UnsupportedPartition               ShortDesc = "Unsupported PARTITION clause"
	UnsupportedPrecision               ShortDesc = "Unsupported precision"
	UnsupportedRowGUIDCOL              ShortDesc = "ROWGUIDCOL not supported"
	UnsupportedSequenceOption          ShortDesc = "Unsupported sequence option"
	UnsupportedTableConnection         ShortDesc = "Unsupported CONNECTION constraint"
	UnsupportedTablePartitioning       ShortDesc = "Unsupported table partitioning"
	UnsupportedTablePhysicalProperties ShortDesc = "Unsupported table physical properties"
	UnsupportedTinyInt                 ShortDesc = "TINYINT not supported"
	Varchar2BytesLimit                 ShortDesc = "VARCHAR2 with BYTES limit"
)

var (
	UnsupportedCharset    = func(c string) ShortDesc { return fmt.Sprintf("charset %s not supported", c) }
	UnsupportedCollate    = func(c string) ShortDesc { return fmt.Sprintf("collate %s not supported", c) }
	UnsupportedColumnType = func(c string) ShortDesc { return fmt.Sprintf("%s %s", UnsupportedColumnTypeRaw, c) }
	UnsupportedDDL        = func(c string) ShortDesc {
		return fmt.Sprintf("translation for %s unavailable", c)
	}
)
