package mysqlconv

import (
	"github.com/cockroachdb/errors"
	"github.com/lib/pq/oid"
)

func DataTypeToOID(dataType, columnType string) (oid.Oid, error) {
	switch dataType {
	case "integer", "int", "mediumint":
		return oid.T_int4, nil
	case "smallint", "tinyint":
		return oid.T_int2, nil
	case "bigint":
		return oid.T_int8, nil
	case "decimal", "numeric":
		return oid.T_numeric, nil
	case "float":
		return oid.T_float4, nil
	case "double":
		return oid.T_float8, nil
	case "bit":
		return oid.T_varbit, nil
	case "date":
		return oid.T_date, nil
	case "datetime":
		return oid.T_timestamp, nil
	case "timestamp":
		return oid.T_timestamptz, nil
	case "time":
		return oid.T_time, nil
	case "char":
		return oid.T_varchar, nil
	case "varchar":
		return oid.T_varchar, nil
	case "binary":
		return oid.T_bytea, nil
	case "varbinary":
		return oid.T_bytea, nil
	case "blob", "text", "mediumtext", "longtext":
		return oid.T_text, nil
	case "json":
		return oid.T_jsonb, nil
	case "enum":
		return oid.T_anyenum, nil
	case "set":
		return 0, errors.Newf("set not yet handled")
	default:
		return 0, errors.Newf("unhandled data type %s, column type %s", dataType, columnType)
	}
}
