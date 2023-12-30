package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
)

func isCloudStore(store string) bool {
	return store == "AWS" || store == "GCP"
}

func getFetchCmdFromPayload(payload *moltservice.CreateFetchPayload) string {
	cmd := MOLTFetchCommand
	cmd = fmt.Sprintf(`%s \
	--source %s \
	--target %s \
`, cmd, payload.SourceConn, payload.TargetConn)

	// mode
	if payload.Mode == "DIRECT_COPY" {
		cmd = fmt.Sprintf("%s	--direct-copy", cmd)
	} else if payload.Mode == "COPY_FROM" {
		cmd = fmt.Sprintf("%s	--live", cmd)
	}
	cmd = fmt.Sprintf("%s \\\n", cmd)

	// intermediate store
	if payload.Mode != "DIRECT_COPY" {
		if payload.Store == "Local" {
			cmd = fmt.Sprintf(`%s	--local-path %s \
	--local-path-listen-addr %s \
	--local-path-crdb-access-addr %s`, cmd, payload.LocalPath, payload.LocalPathListenAddress, payload.LocalPathCrdbAddress)
		} else if isCloudStore(payload.Store) {
			bucketNameFlag := "--gcp-bucket"
			if payload.Store == "AWS" {
				bucketNameFlag = "--s3-bucket"
			}

			cmd = fmt.Sprintf("%s %s %s", cmd, bucketNameFlag, payload.BucketName)

			if strings.TrimSpace(payload.BucketPath) != "" {
				cmd = fmt.Sprintf("%s --bucket-path %s", cmd, payload.BucketPath)
			}
		}

		if payload.CleanupIntermediaryStore {
			cmd = fmt.Sprintf("%s --cleanup", cmd)
		}
		cmd = fmt.Sprintf("%s \\\n", cmd)
	}

	// task level setting
	cmd = fmt.Sprintf("%s	--compression %s", cmd, payload.Compression)
	if strings.TrimSpace(payload.LogFile) != "" {
		cmd = fmt.Sprintf("%s --log-file %s", cmd, payload.LogFile)
	}
	if payload.Truncate {
		cmd = fmt.Sprintf("%s --truncate", cmd)
	}
	cmd = fmt.Sprintf("%s \\\n", cmd)

	// performance tuning
	if strings.TrimSpace(payload.PgLogicalSlotName) != "" {
		cmd = fmt.Sprintf("%s --pg-logical-replication-slot-name %s \\\n", cmd, payload.PgLogicalSlotName)
	}

	if strings.TrimSpace(payload.PgLogicalPlugin) != "" {
		cmd = fmt.Sprintf("%s --pg-logical-replication-slot-plugin %s \\\n", cmd, payload.PgLogicalPlugin)
	}

	if payload.PgDropSlot {
		cmd = fmt.Sprintf("%s --pg-logical-replication-slot-drop-if-exists", cmd)
	}

	return strings.TrimSuffix(strings.TrimSpace(cmd), "\\")
}

func (m *moltService) CreateFetchTask(
	ctx context.Context, payload *moltservice.CreateFetchPayload,
) (res moltservice.FetchAttemptID, err error) {
	fmt.Println(getFetchCmdFromPayload(payload))
	return 1, nil
}
