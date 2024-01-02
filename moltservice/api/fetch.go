package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
)

type FetchStatus string

const (
	FetchStatusInProgress = "IN_PROGRESS"
	FetchStatusSuccess    = "SUCCESS"
	FetchStatusFailure    = "FAILURE"
)

type FetchDetail struct {
	ID           moltservice.FetchAttemptID `json:"id"`
	LogTimestamp time.Time                  `json:"time"`
	Status       FetchStatus                `json:"status"`
	Note         string                     `json:"note"`
	LogFile      string                     `json:"-"`
	StartedAt    time.Time                  `json:"started_at"`
	FinishedAt   time.Time                  `json:"finished_at"`

	// TODO: add tracking stats to fetch detail to report on.
}

const staticTaskLog = "artifacts/task.log"

func isCloudStore(store string) bool {
	return store == "AWS" || store == "GCP"
}

func getFetchArgsFromPayload(payload *moltservice.CreateFetchPayload) []string {
	commandList := []string{fetch, "--source", payload.SourceConn, "--target", payload.TargetConn}

	// mode
	if payload.Mode == "DIRECT_COPY" {
		commandList = append(commandList, "--direct-copy")
	} else if payload.Mode == "COPY_FROM" {
		commandList = append(commandList, "--live")
	}

	// intermediate store
	if payload.Mode != "DIRECT_COPY" {
		if payload.Store == "Local" {
			commandList = append(commandList, "local-path", payload.LocalPath,
				"--local-path-listen-addr", payload.LocalPathListenAddress,
				"--local-path-crdb-access-addr", payload.LocalPathCrdbAddress,
			)
		} else if isCloudStore(payload.Store) {
			bucketNameFlag := "--gcp-bucket"
			if payload.Store == "AWS" {
				bucketNameFlag = "--s3-bucket"
			}

			commandList = append(commandList, bucketNameFlag, payload.BucketName)
			if strings.TrimSpace(payload.BucketPath) != "" {
				commandList = append(commandList, "--bucket-path", payload.BucketPath)
			}
		}

		if payload.CleanupIntermediaryStore {
			commandList = append(commandList, "--cleanup")
		}
	}

	// task level setting
	commandList = append(commandList, "--compression", payload.Compression)
	if strings.TrimSpace(payload.LogFile) != "" {
		commandList = append(commandList, "--log-file", payload.LogFile)
	}
	if payload.Truncate {
		commandList = append(commandList, "--truncate")
	}

	// performance tuning
	if payload.NumFlushRows > 0 {
		commandList = append(commandList, "--flush-rows", strconv.Itoa(payload.NumFlushRows))
	}

	if payload.NumFlushBytes > 0 {
		commandList = append(commandList, "--flush-size", strconv.Itoa(payload.NumFlushBytes))
	}

	if payload.NumConcurrentTables > 0 {
		commandList = append(commandList, "--concurrency", strconv.Itoa(payload.NumConcurrentTables))
	}

	if payload.NumBatchRowsExport > 0 {
		commandList = append(commandList, "--row-batch-size", strconv.Itoa(payload.NumBatchRowsExport))
	}

	// replication
	if strings.TrimSpace(payload.PgLogicalSlotName) != "" {
		commandList = append(commandList, "--pg-logical-replication-slot-name", payload.PgLogicalSlotName)
	}

	if strings.TrimSpace(payload.PgLogicalPlugin) != "" {
		commandList = append(commandList, "--pg-logical-replication-slot-plugin", payload.PgLogicalPlugin)
	}

	if payload.PgDropSlot {
		commandList = append(commandList, "--pg-logical-replication-slot-drop-if-exists")
	}

	return commandList
}

// Key issues TODO (rluu):
// 1. Stderr is not coming out properly to the task log file
// 3. We need to figure out how to stream logs
func (m *moltService) CreateFetchTask(
	ctx context.Context, payload *moltservice.CreateFetchPayload,
) (res moltservice.FetchAttemptID, err error) {
	// TODO: figure out more elegant way to override this later. For now, make everything reference static task.log
	// with prepended timestamp. Also need to figure out a better unique id that has sorting.
	// TODO: write these statuses to the database so that we can actually see what's running. Needs system startup logic.
	id := moltservice.FetchAttemptID(time.Now().Unix())
	payload.LogFile = fmt.Sprintf("%s-%d", staticTaskLog, id)

	args := getFetchArgsFromPayload(payload)
	fetchDetail := FetchDetail{
		ID:           id,
		LogTimestamp: time.Now(),
		LogFile:      payload.LogFile,
		Status:       FetchStatusInProgress,
		StartedAt:    time.Now(),
	}

	m.fetchState.Lock()
	m.fetchState.orderedIdList = append(m.fetchState.orderedIdList, id)
	m.fetchState.idToRun[id] = fetchDetail
	m.fetchState.Unlock()

	go func(detail FetchDetail) {
		if m.debugEnabled {
			fmt.Println("Args: ", args)
		}

		// Write the beginning status to the file.
		if err := writeFetchDetail(fetchDetail, payload.LogFile); err != nil {
			m.logger.Err(err).Send()
		}

		out, err := exec.Command(MOLTCommand, args...).Output()

		// Update fetch detail with the latest details.
		fetchDetail.LogTimestamp = time.Now()
		fetchDetail.Status = FetchStatusSuccess
		fetchDetail.FinishedAt = time.Now()

		if err != nil {
			fetchDetail.Status = FetchStatusFailure
			m.logger.Err(err).Send()
		}
		if m.debugEnabled {
			fmt.Println(string(out))
		}

		// Write the ending status to the file.
		if err := writeFetchDetail(fetchDetail, payload.LogFile); err != nil {
			m.logger.Err(err).Send()
		}

		// Update the map.
		m.fetchState.Lock()
		m.fetchState.idToRun[id] = fetchDetail
		m.fetchState.Unlock()
	}(fetchDetail)

	return moltservice.FetchAttemptID(id), err
}

func writeFetchDetail(detail FetchDetail, logFile string) error {
	jsonData, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	jsonData = append(jsonData, '\n')

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}
