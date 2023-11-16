package mysqlconv

import (
	"github.com/cockroachdb/errors"
	"github.com/lib/pq/oid"
)

func DataTypeToOID(dataType, columnType string) oid.Oid {
	switch dataType {
	case "integer", "int", "mediumint":
		return oid.T_int4
	case "smallint", "tinyint":
		return oid.T_int2
	case "bigint":
		return oid.T_int8
	case "decimal", "numeric":
		return oid.T_numeric
	case "float":
		return oid.T_float4
	case "double":
		return oid.T_float8
	case "bit":
		return oid.T_varbit
	case "date":
		return oid.T_date
	case "datetime":
		return oid.T_timestamp
	case "timestamp":
		return oid.T_timestamptz
	case "time":
		return oid.T_time
	case "char":
		return oid.T_varchar
	case "varchar":
		return oid.T_varchar
	case "binary":
		return oid.T_bytea
	case "varbinary":
		return oid.T_bytea
	case "blob", "text", "mediumtext", "longtext":
		return oid.T_text
	case "json":
		return oid.T_jsonb
	case "enum":
		return oid.T_anyenum
	case "set":
		panic(errors.Newf("enums not yet handled"))
	default:
		panic(errors.Newf("unhandled data type %s, column type %s", dataType, columnType))
	}
}
