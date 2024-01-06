package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
)

type VerifyStatus string

const (
	VerifyStatusInProgress = VerifyStatus("IN_PROGRESS")
	VerifyStatusSuccess    = VerifyStatus("SUCCESS")
	VerifyStatusFailure    = VerifyStatus("FAILURE")
)

const numVerifyLogLines = 1000

type VerifyDetail struct {
	RunName      string                      `json:"run_name"`
	ID           moltservice.VerifyAttemptID `json:"id"`
	LogTimestamp time.Time                   `json:"time"`
	Status       VerifyStatus                `json:"status"`
	Note         string                      `json:"note"`
	LogFile      string                      `json:"-"`
	StartedAt    time.Time                   `json:"started_at"`
	FinishedAt   time.Time                   `json:"finished_at"`
	FetchID      moltservice.FetchAttemptID  `json:"fetch_id"`
}

func (vd *VerifyDetail) mapToResponse() *moltservice.VerifyRun {
	return &moltservice.VerifyRun{
		ID:         int(vd.ID),
		Name:       vd.RunName,
		Status:     string(vd.Status),
		StartedAt:  normalizeTimestamp(vd.StartedAt),
		FinishedAt: normalizeTimestamp(vd.FinishedAt),
		FetchID:    int(vd.FetchID),
	}
}

func (v *VerifyDetail) mapToDetailedResponse(
	stats *moltservice.VerifyStatsDetailed, mismatchLogs []*moltservice.VerifyMismatch,
) *moltservice.VerifyRunDetailed {
	return &moltservice.VerifyRunDetailed{
		ID:         int(v.ID),
		Name:       v.RunName,
		Status:     string(v.Status),
		StartedAt:  normalizeTimestamp(v.StartedAt),
		FinishedAt: normalizeTimestamp(v.FinishedAt),
		Stats:      stats,
		Mismatches: mismatchLogs,
		FetchID:    int(v.FetchID),
	}
}

type VerifySummaryLog struct {
	NumTables             int    `json:"-"`
	Schema                string `json:"table_schema"`
	TableName             string `json:"table_name"`
	NumRows               int    `json:"num_truth_rows"`
	NumSuccess            int    `json:"num_success"`
	NumConditionalSuccess int    `json:"num_conditional_success"`
	NumMissing            int    `json:"num_missing"`
	NumMismatch           int    `json:"num_mismatch"`
	NumExtraneous         int    `json:"num_extraneous"`
	NumLiveRetry          int    `json:"num_live_retry"`
	NumColumnMismatch     int    `json:"num_column_mismatch"`
}

type VerificationCompleteLog struct {
	NetDurationMS float64 `json:"net_duration_ms"`
	NetDuration   string  `json:"net_duration"`
}

type VerifyMismatchLog struct {
	Time      string `json:"time"`
	Level     string `json:"level"`
	Message   string `json:"-"`
	Schema    string `json:"table_schema"`
	TableName string `json:"table_name"`
	Type      string `json:"message"`
}

func (v *VerifyMismatchLog) mapToResponse() (*moltservice.VerifyMismatch, error) {
	parsedTime, err := time.Parse(time.RFC3339, v.Time)
	if err != nil {
		return nil, err
	}

	return &moltservice.VerifyMismatch{
		Timestamp: int(parsedTime.Unix()),
		Level:     v.Level,
		Message:   v.Message,
		Schema:    v.Schema,
		Table:     v.TableName,
		Type:      v.Type,
	}, nil
}

const staticVerifyLog = "artifacts/verify.log"

func constructVerifyArgs(source, target, logFile string) []string {
	commandList := []string{verify, "--source", source, "--target", target, "--log-file", logFile}
	return commandList
}

func (m *moltService) CreateVerifyTaskFromFetch(
	ctx context.Context, payload *moltservice.CreateVerifyTaskFromFetchPayload,
) (res moltservice.VerifyAttemptID, err error) {
	run, ok := m.fetchState.idToRun[moltservice.FetchAttemptID(payload.ID)]
	if !ok {
		return -1, errors.Newf("failed to find fetch task with id %d", payload.ID)
	}

	verifyId := moltservice.VerifyAttemptID(time.Now().Unix())
	verifyLogFile := fmt.Sprintf("%s-%d", staticVerifyLog, verifyId)

	args := constructVerifyArgs(run.ConfigurationPayload.SourceConn, run.ConfigurationPayload.TargetConn, verifyLogFile)

	verifyDetail := VerifyDetail{
		RunName:      fmt.Sprintf("%s-%d", run.RunName, len(run.VerifyIDs)+1),
		ID:           verifyId,
		LogTimestamp: time.Now(),
		LogFile:      verifyLogFile,
		Status:       VerifyStatusInProgress,
		StartedAt:    time.Now(),
		FetchID:      run.ID,
	}

	// Update verify state.
	m.verifyState.Lock()
	m.verifyState.orderedIdList = append(m.verifyState.orderedIdList, verifyId)
	m.verifyState.idToRun[verifyId] = verifyDetail
	m.verifyState.Unlock()

	// Update fetch state.
	m.fetchState.Lock()
	run.VerifyIDs = append(run.VerifyIDs, verifyId)
	m.fetchState.idToRun[run.ID] = run
	m.fetchState.Unlock()

	go func() {
		if m.debugEnabled {
			fmt.Println("Args: ", args)
		}

		// Write the beginning status to the file.
		if err := writeDetail(verifyDetail, verifyLogFile); err != nil {
			m.logger.Err(err).Send()
		}

		out, err := exec.Command(MOLTCommand, args...).CombinedOutput()

		// Update with the latest details.
		verifyDetail.LogTimestamp = time.Now()
		verifyDetail.Status = VerifyStatusSuccess
		verifyDetail.FinishedAt = time.Now()

		if err != nil {
			verifyDetail.Status = VerifyStatusFailure
			m.logger.Err(err).Send()
			errMessage := string(out)
			log := Log{
				Time:    time.Now().Format(time.RFC3339),
				Level:   "error",
				Message: errMessage,
			}
			writeDetail(log, verifyLogFile)
		}

		// Write the ending status to the file.
		if err := writeDetail(verifyDetail, verifyLogFile); err != nil {
			m.logger.Err(err).Send()
		}

		// Update the map.
		m.verifyState.Lock()
		m.verifyState.idToRun[verifyId] = verifyDetail
		m.verifyState.Unlock()

		if m.debugEnabled {
			fmt.Println(m.verifyState.idToRun[verifyId])
			fmt.Println(m.fetchState.idToRun[run.ID])
		}
	}()

	return verifyId, nil
}

func (m *moltService) getVerifyTasks(
	idList []moltservice.VerifyAttemptID, canBeEmpty bool,
) (res []*moltservice.VerifyRun, err error) {
	orderedIdList := idList

	if len(idList) == 0 && !canBeEmpty {
		orderedIdList = m.verifyState.orderedIdList
	}

	m.verifyState.Lock()
	defer m.verifyState.Unlock()
	verifyRuns := []*moltservice.VerifyRun{}

	for i := len(orderedIdList) - 1; i >= 0; i-- {
		id := orderedIdList[i]
		run, ok := m.verifyState.idToRun[id]
		if !ok {
			return nil, errors.Newf("failed to get fetch run with id %s", id)
		}

		runResp := run.mapToResponse()
		verifyRuns = append(verifyRuns, runResp)
	}

	return verifyRuns, nil
}

func (m *moltService) GetVerifyTasks(
	ctx context.Context,
) (res []*moltservice.VerifyRun, err error) {
	return m.getVerifyTasks([]moltservice.VerifyAttemptID{}, false /*canBeEmpty*/)
}

func (m *moltService) GetSpecificVerifyTask(
	ctx context.Context, payload *moltservice.GetSpecificVerifyTaskPayload,
) (res *moltservice.VerifyRunDetailed, err error) {
	run, ok := m.verifyState.idToRun[moltservice.VerifyAttemptID(payload.ID)]
	if !ok {
		return nil, errors.Newf("failed to find fetch task with id %d", payload.ID)
	}

	lines, err := readNLines(run.LogFile, numVerifyLogLines)
	if err != nil {
		return nil, err
	}

	stats, err := m.extractVerifyStats(lines)
	if err != nil {
		return nil, err
	}

	mismatches, err := m.extractVerifyMismatches(lines)
	if err != nil {
		return nil, err
	}

	return run.mapToDetailedResponse(stats, mismatches), nil
}

func (m *moltService) extractVerifyStats(lines []string) (*moltservice.VerifyStatsDetailed, error) {
	stats := &moltservice.VerifyStatsDetailed{}

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		if strings.Contains(line, "verification complete") {
			var logLine VerificationCompleteLog
			err := json.Unmarshal([]byte(line), &logLine)
			if err != nil {
				m.logger.Err(err).Send()
				continue
			}
			stats.NetDurationMs = logLine.NetDurationMS
		}

		// TODO (rluu): in the future don't assume that tables can't be duplicated
		// Check a map first to see if the table + schema is present. For now
		// can just assume people are running the most base verify mode.
		if strings.Contains(line, "finished row verification") {
			var logLine VerifySummaryLog
			err := json.Unmarshal([]byte(line), &logLine)
			if err != nil {
				m.logger.Err(err).Send()
				continue
			}
			stats.NumTables++
			stats.NumTruthRows += logLine.NumTables
			stats.NumSuccess += logLine.NumSuccess
			stats.NumConditionalSuccess += logLine.NumConditionalSuccess
			stats.NumMismatch += logLine.NumMismatch
			stats.NumMissing += logLine.NumMissing
			stats.NumExtraneous += logLine.NumExtraneous
			stats.NumLiveRetry += logLine.NumLiveRetry
			stats.NumColumnMismatch += logLine.NumColumnMismatch
		}
	}

	return stats, nil
}

func (m *moltService) extractVerifyMismatches(
	lines []string,
) ([]*moltservice.VerifyMismatch, error) {
	mismatches := make([]*moltservice.VerifyMismatch, 0)

	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		if strings.Contains(line, `"type":"data"`) {
			var logLine VerifyMismatchLog
			err := json.Unmarshal([]byte(line), &logLine)
			if err != nil {
				m.logger.Err(err).Send()
				continue
			}

			mismatch, err := logLine.mapToResponse()
			if err != nil {
				return nil, err
			}

			mismatch.Message = line
			mismatches = append(mismatches, mismatch)
		}

	}

	return mismatches, nil
}
