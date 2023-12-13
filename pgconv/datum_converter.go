package pgconv

import (
	"fmt"
	"time"

	"github.com/cockroachdb/apd/v3"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/json"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/timeofday"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/timeutil/pgdate"
	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/parsectx"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq/oid"
)

func ConvertRowValue(typMap *pgtype.Map, val any, typOID oid.Oid) (tree.Datum, error) {
	if val == nil {
		return tree.DNull, nil
	}
	if _, isArray := types.ArrayOids[typOID]; isArray {
		arrayType := types.OidToType[typOID]
		ret := tree.NewDArray(arrayType.ArrayContents())
		// Only worry about 1D arrays for now.
		for arrIdx, arr := range val.([]interface{}) {
			elem, err := ConvertRowValue(typMap, arr, arrayType.ArrayContents().Oid())
			if err != nil {
				return nil, errors.Wrapf(err, "error converting array element %d", arrIdx)
			}
			if err := ret.Append(elem); err != nil {
				return nil, errors.Wrapf(err, "error appending array element %d", arrIdx)
			}
		}
		return ret, nil
	}

	switch typOID {
	case pgtype.BoolOID:
		// Convert the boolean to an integer so that it's reasonable to compare with
		// another TINYINT for MySQL and other dialects that don't have a BOOL.
		intVal := 0
		if val.(bool) {
			intVal = 1
		}

		return tree.ParseDInt(fmt.Sprintf("%d", intVal))
	case pgtype.QCharOID:
		return tree.NewDString(fmt.Sprintf("%c", val.(int32))), nil
	case pgtype.VarcharOID, pgtype.TextOID, pgtype.BPCharOID:
		return tree.NewDString(val.(string)), nil
	case pgtype.NameOID:
		return tree.NewDName(val.(string)), nil
	case pgtype.Float4OID:
		return tree.NewDFloat(tree.DFloat(val.(float32))), nil
	case pgtype.Float8OID:
		return tree.NewDFloat(tree.DFloat(val.(float64))), nil
	case pgtype.Int2OID:
		return tree.NewDInt(tree.DInt(val.(int16))), nil
	case pgtype.Int4OID:
		return tree.NewDInt(tree.DInt(val.(int32))), nil
	case pgtype.Int8OID:
		return tree.NewDInt(tree.DInt(val.(int64))), nil
	case pgtype.OIDOID:
		return tree.NewDOid(oid.Oid(val.(uint32))), nil
	case pgtype.JSONOID, pgtype.JSONBOID:
		j, err := json.MakeJSON(val)
		if err != nil {
			return nil, errors.Wrapf(err, "error decoding json for %v", val)
		}
		return tree.NewDJSON(j), nil
	case pgtype.UUIDOID:
		b := val.([16]uint8)
		u, err := uuid.FromBytes(b[:])
		if err != nil {
			return nil, errors.Wrapf(err, "error decoding UUID %v", val)
		}
		return tree.NewDString(u.String()), nil
	case pgtype.TimestampOID:
		return tree.MakeDTimestamp(val.(time.Time), time.Microsecond)
	case pgtype.TimestamptzOID:
		return tree.MakeDTimestampTZ(val.(time.Time).UTC(), time.Microsecond)
	case pgtype.TimeOID:
		if val.(pgtype.Time).Microseconds == 24*60*60*1000000 {
			return tree.MakeDTime(timeofday.Time2400), nil
		}
		return tree.MakeDTime(timeofday.FromInt(val.(pgtype.Time).Microseconds)), nil
	case pgtype.DateOID:
		d, err := pgdate.MakeDateFromTime(val.(time.Time))
		if err != nil {
			return nil, errors.Wrapf(err, "error converting date %v", val)
		}
		return tree.NewDDate(d), nil
	case pgtype.ByteaOID:
		return tree.NewDBytes(tree.DBytes(val.([]byte))), nil
	case oid.T_timetz: // does not exist in pgtype.
		d, _, err := tree.ParseDTimeTZ(parsectx.ParseContext, val.(string), time.Microsecond)
		return d, err
	case pgtype.NumericOID:
		return convertNumeric(val.(pgtype.Numeric))
	case pgtype.BitOID, pgtype.VarbitOID:
		// This can be a lot more efficient, but we don't have the right abstractions.
		val := val.(pgtype.Bits)
		var buf []byte
		for i := int32(0); i < val.Len; i++ {
			byteIdx := i / 8
			bitMask := byte(128 >> byte(i%8))
			char := byte('0')
			if val.Bytes[byteIdx]&bitMask > 0 {
				char = '1'
			}
			buf = append(buf, char)
		}
		return tree.ParseDBitArray(string(buf))
	}
	typ, ok := typMap.TypeForOID(uint32(typOID))
	if !ok {
		return nil, errors.AssertionFailedf("value %v (%T) of type OID %d not initialised", val, val, typOID)
	}
	switch cdc := typ.Codec.(type) {
	case *pgtype.EnumCodec:
		return tree.NewDString(val.(string)), nil
	case *pgtype.ArrayCodec:
		// Support enum arrays by casting to string.
		if _, ok := cdc.ElementType.Codec.(*pgtype.EnumCodec); ok {
			ret := tree.NewDArray(types.String)
			for arrIdx, arr := range val.([]interface{}) {
				elem, err := ConvertRowValue(typMap, arr, oid.T_text)
				if err != nil {
					return nil, errors.Wrapf(err, "error converting array element %d", arrIdx)
				}
				if err := ret.Append(elem); err != nil {
					return nil, errors.Wrapf(err, "error appending array element %d", arrIdx)
				}
			}
			return ret, nil
		}

	}
	return nil, errors.AssertionFailedf("value %v (%T) of type OID %d not yet translatable", val, val, typOID)
}

func convertNumeric(val pgtype.Numeric) (*tree.DDecimal, error) {
	if val.NaN {
		return tree.ParseDDecimal("NaN")
	} else if val.InfinityModifier == pgtype.Infinity {
		return tree.ParseDDecimal("Inf")
	} else if val.InfinityModifier == pgtype.NegativeInfinity {
		return tree.ParseDDecimal("-Inf")
	}
	return &tree.DDecimal{Decimal: *apd.New(val.Int.Int64(), val.Exp)}, nil
}

func ConvertRowValues(typMap *pgtype.Map, vals []any, typOIDs []oid.Oid) (tree.Datums, error) {
	ret := make(tree.Datums, len(vals))
	if len(vals) != len(typOIDs) {
		return nil, errors.AssertionFailedf("val length != oid length: %v vs %v", vals, typOIDs)
	}
	for i := range vals {
		var err error
		if ret[i], err = ConvertRowValue(typMap, vals[i], typOIDs[i]); err != nil {
			return nil, err
		}
	}
	return ret, nil
}
