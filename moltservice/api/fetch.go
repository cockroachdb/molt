package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
)

type FetchStatus string

const (
	FetchStatusInProgress = "IN_PROGRESS"
	FetchStatusSuccess    = "SUCCESS"
	FetchStatusFailure    = "FAILURE"
)

const (
	numLogLines = 100
)

type ExportSummaryLog struct {
	NumRows        int    `json:"num_rows"`
	ExportDuration string `json:"export_duration"`
}

type OverallSummaryLog struct {
	NumTables int    `json:"num_tables"`
	CDCCursor string `json:"cdc_cursor"`
}

type CompletionPercentLog struct {
	Completion int `json:"completion"`
}

type Log struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

func (l *Log) mapToResponse() (*moltservice.Log, error) {
	parsedTime, err := time.Parse(time.RFC3339, l.Time)
	if err != nil {
		return nil, err
	}

	return &moltservice.Log{
		Timestamp: int(parsedTime.Unix()),
		Level:     l.Level,
		Message:   l.Message,
	}, nil
}

type FetchDetail struct {
	RunName      string                     `json:"run_name"`
	ID           moltservice.FetchAttemptID `json:"id"`
	LogTimestamp time.Time                  `json:"time"`
	Status       FetchStatus                `json:"status"`
	Note         string                     `json:"note"`
	LogFile      string                     `json:"-"`
	StartedAt    time.Time                  `json:"started_at"`
	FinishedAt   time.Time                  `json:"finished_at"`

	// TODO: add tracking stats to fetch detail to report on.
}

func (fd *FetchDetail) mapToResponse() *moltservice.FetchRun {
	return &moltservice.FetchRun{
		ID:         int(fd.ID),
		Name:       fd.RunName,
		Status:     string(fd.Status),
		StartedAt:  normalizeTimestamp(fd.StartedAt),
		FinishedAt: normalizeTimestamp(fd.FinishedAt),
	}
}

func (fd *FetchDetail) mapToDetailedResponse(
	stats *moltservice.FetchStatsDetailed, logs []*moltservice.Log,
) *moltservice.FetchRunDetailed {
	return &moltservice.FetchRunDetailed{
		ID:         int(fd.ID),
		Name:       fd.RunName,
		Status:     string(fd.Status),
		StartedAt:  normalizeTimestamp(fd.StartedAt),
		FinishedAt: normalizeTimestamp(fd.FinishedAt),
		Stats:      stats,
		Logs:       logs,
	}
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
		RunName:      payload.Name,
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

		if m.debugEnabled {
			fmt.Println(m.fetchState.idToRun[id])
		}
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

func (m *moltService) GetFetchTasks(ctx context.Context) (res []*moltservice.FetchRun, err error) {
	fetchRuns := []*moltservice.FetchRun{}

	for i := len(m.fetchState.orderedIdList) - 1; i >= 0; i-- {
		id := m.fetchState.orderedIdList[i]
		run, ok := m.fetchState.idToRun[id]
		if !ok {
			return nil, errors.Newf("failed to get fetch run with id %s", id)
		}

		runResp := run.mapToResponse()
		fetchRuns = append(fetchRuns, runResp)
	}

	return fetchRuns, nil
}

func (m *moltService) GetSpecificFetchTask(
	ctx context.Context, payload *moltservice.GetSpecificFetchTaskPayload,
) (res *moltservice.FetchRunDetailed, err error) {
	run, ok := m.fetchState.idToRun[moltservice.FetchAttemptID(payload.ID)]
	if !ok {
		return nil, errors.Newf("failed to find fetch task with id %d", payload.ID)
	}

	lines, err := readNLines(run.LogFile, numLogLines)
	if err != nil {
		return nil, err
	}

	// TODO (rluu): extract stats later
	stats, err := m.extractStats(lines)
	if err != nil {
		return nil, err
	}

	// Extract log lines.
	logLines, err := m.extractLogLines(lines)
	if err != nil {
		return nil, err
	}

	runResp := run.mapToDetailedResponse(stats, logLines)
	return runResp, nil
}

func readNLines(filePath string, numLines int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0, numLines)

	for scanner.Scan() && len(lines) < numLines {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// TODO (rluu): add logic for percentage estimation.
// This will require getting an idea of percentage estimate per table, which is non trivial.
// TODO (rluu): add the following exposed fields:
// - CDC cursor
// - Import duration
// - Export duration
// - Net duration (if applicable)
func (m *moltService) extractStats(lines []string) (*moltservice.FetchStatsDetailed, error) {
	stats := &moltservice.FetchStatsDetailed{}
	foundExportSummary := false
	foundNumTables := false

	for i := len(lines) - 1; i >= 0; i-- {
		// Means we found all the stats we want.
		if foundExportSummary && foundNumTables {
			break
		}

		line := lines[i]

		if strings.Contains(line, "fetch complete") && !foundNumTables {
			var logLine OverallSummaryLog
			err := json.Unmarshal([]byte(line), &logLine)
			if err != nil {
				m.logger.Err(err).Send()
				continue
			}
			foundNumTables = true
			stats.NumTables = logLine.NumTables
			// Fetch complete means that this completed successfully.
			stats.PercentComplete = "100"
		}

		if strings.Contains(line, "data extraction from source complete") && !foundExportSummary {
			var logLine ExportSummaryLog
			err := json.Unmarshal([]byte(line), &logLine)
			if err != nil {
				m.logger.Err(err).Send()
				continue
			}
			foundExportSummary = true
			stats.NumRows = logLine.NumRows
		}
	}

	return stats, nil
}

func (m *moltService) extractLogLines(lines []string) ([]*moltservice.Log, error) {
	logLines := []*moltservice.Log{}
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		// skip the moltservice specific status lines.
		if strings.Contains(line, `"run_name"`) {
			continue
		}

		var logLine Log
		err := json.Unmarshal([]byte(line), &logLine)
		if err != nil {
			m.logger.Err(err).Send()
			continue
		}

		logLine.Message = line
		finalLine, err := logLine.mapToResponse()
		if err != nil {
			m.logger.Err(err).Send()
			continue
		}

		logLines = append(logLines, finalLine)
	}

	return logLines, nil
}
