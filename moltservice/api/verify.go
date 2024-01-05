package api

import (
	"context"
	"fmt"
	"os/exec"
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

	// TODO: add tracking stats to fetch detail to report on.
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
		RunName:      run.RunName,
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

		out, err := exec.Command(MOLTCommand, args...).Output()

		// Update with the latest details.
		verifyDetail.LogTimestamp = time.Now()
		verifyDetail.Status = VerifyStatusSuccess
		verifyDetail.FinishedAt = time.Now()

		if err != nil {
			verifyDetail.Status = VerifyStatusFailure
			m.logger.Err(err).Send()
		}

		if m.debugEnabled {
			fmt.Println(string(out))
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

// TODO: get detailed verify along with all errors, mismatches, etc.
