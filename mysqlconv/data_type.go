package mysqlconv

import (
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/utils/typeconv"
	"github.com/lib/pq/oid"
	"github.com/rs/zerolog"
)

func DataTypeToOID(dataType string, logger zerolog.Logger) (oid.Oid, error) {
	fakeMeta := typeconv.NewMySQLColumnMeta()
	t, tfs := fakeMeta.ToDefaultCRDBType(dataType, "")
	for _, tf := range tfs {
		if tf.Blocking {
			err := errors.Newf("cannot get the corresponding oid for type %s: %s", dataType, tf.ShortDescription)
			logger.Err(err)
			return 0, err
		}
		logger.Warn().Msgf(tf.Message)
	}
	return t.Oid(), nil
}
