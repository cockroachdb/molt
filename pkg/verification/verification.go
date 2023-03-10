package verification

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/types"
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/pkg/ctxgroup"
	"github.com/cockroachdb/molt/pkg/dbconn"
	"github.com/lib/pq/oid"
)

func init() {
	// Inject JSON as a OidToType.
	types.OidToType[oid.T_json] = types.Jsonb
	types.OidToType[oid.T__json] = types.MakeArray(types.Jsonb)
}

func (tm TableMetadata) Compare(o TableMetadata) int {
	if c := strings.Compare(string(tm.Schema), string(o.Schema)); c != 0 {
		return c
	}
	return strings.Compare(string(tm.Table), string(o.Table))
}

func (tm TableMetadata) Less(o TableMetadata) bool {
	return tm.Compare(o) < 0
}

const DefaultConcurrency = 8
const DefaultRowBatchSize = 1000
const DefaultTableSplits = 8

type VerifyOpt func(*verifyOpts)

type WorkFunc func(
	ctx context.Context,
	conns []dbconn.Conn,
	table TableShard,
	rowBatchSize int,
	reporter Reporter,
) error

type verifyOpts struct {
	concurrency  int
	rowBatchSize int
	tableSplits  int
	// TODO: better abstraction for this.
	workFunc WorkFunc
}

func WithConcurrency(c int) VerifyOpt {
	return func(o *verifyOpts) {
		o.concurrency = c
	}
}

func WithRowBatchSize(c int) VerifyOpt {
	return func(o *verifyOpts) {
		o.rowBatchSize = c
	}
}

func WithTableSplits(c int) VerifyOpt {
	return func(o *verifyOpts) {
		o.tableSplits = c
	}
}

func WithWorkFunc(c WorkFunc) VerifyOpt {
	return func(o *verifyOpts) {
		o.workFunc = c
	}
}

// Verify verifies the given connections have matching tables and contents.
func Verify(
	ctx context.Context, conns []dbconn.Conn, reporter Reporter, inOpts ...VerifyOpt,
) error {
	opts := verifyOpts{
		concurrency:  DefaultConcurrency,
		rowBatchSize: DefaultRowBatchSize,
		tableSplits:  DefaultTableSplits,
		workFunc:     CompareRows,
	}
	for _, applyOpt := range inOpts {
		applyOpt(&opts)
	}

	ret, err := verifyDatabaseTables(ctx, conns)
	if err != nil {
		return errors.Wrap(err, "error comparing database tables")
	}

	for _, missingTable := range ret.missingTables {
		reporter.Report(missingTable)
	}
	for _, extraneousTable := range ret.extraneousTables {
		reporter.Report(extraneousTable)
	}

	// Grab columns for each table on both sides.
	tbls, err := verifyCommonTables(ctx, conns, ret.verified)
	if err != nil {
		return err
	}

	// Report mismatching table definitions.
	for _, tbl := range tbls {
		for _, d := range tbl.MismatchingTableDefinitions {
			reporter.Report(d)
		}
	}

	// Compare rows up to the concurrency specified.
	g := ctxgroup.WithContext(ctx)
	workQueue := make(chan TableShard)
	for it := 0; it < opts.concurrency; it++ {
		g.GoCtx(func(ctx context.Context) error {
			for {
				splitTable, ok := <-workQueue
				if !ok {
					return nil
				}
				msg := fmt.Sprintf(
					"starting verify on %s.%s, shard %d/%d",
					splitTable.Schema,
					splitTable.Table,
					splitTable.ShardNum,
					splitTable.TotalShards,
				)
				if splitTable.TotalShards > 1 {
					msg += ", range: ["
					if len(splitTable.StartPKVals) > 0 {
						for i, val := range splitTable.StartPKVals {
							if i > 0 {
								msg += ","
							}
							msg += val.String()
						}
					} else {
						msg += "<beginning>"
					}
					msg += " - "
					if len(splitTable.EndPKVals) > 0 {
						for i, val := range splitTable.EndPKVals {
							if i > 0 {
								msg += ", "
							}
							msg += val.String()
						}
						msg += ")"
					} else {
						msg += "<end>]"
					}
				}
				reporter.Report(StatusReport{
					Info: msg,
				})
				if err := verifyDataWorker(ctx, conns, reporter, opts.rowBatchSize, splitTable, opts.workFunc); err != nil {
					log.Printf("[ERROR] error comparing rows on %s.%s: %v", splitTable.Schema, splitTable.Table, err)
				}
			}
		})
	}
	for _, tbl := range tbls {
		// Ignore tables which cannot be verified.
		if !tbl.RowVerifiable {
			continue
		}

		// Get and first and last of each PK.
		splitTables, err := splitTable(ctx, conns[0], tbl, reporter, opts.tableSplits)
		if err != nil {
			return errors.Wrapf(err, "error splitting tables")
		}
		for _, splitTable := range splitTables {
			workQueue <- splitTable
		}
	}
	close(workQueue)
	return g.Wait()
}

func verifyDataWorker(
	ctx context.Context,
	conns []dbconn.Conn,
	reporter Reporter,
	rowBatchSize int,
	tbl TableShard,
	workFunc WorkFunc,
) error {
	// Copy connections over naming wise, but initialize a new pgx connection
	// for each table.
	workerConns := make([]dbconn.Conn, len(conns))
	for i := range workerConns {
		// Make a copy of i so the worker closes correctly.
		i := i
		var err error
		workerConns[i], err = conns[i].Clone(ctx)
		if err != nil {
			return errors.Wrap(err, "error establishing connection to compare")
		}
		defer func() {
			_ = workerConns[i].Close(ctx)
		}()
	}
	return workFunc(ctx, workerConns, tbl, rowBatchSize, reporter)
}
