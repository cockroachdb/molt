package api

import (
	"sync"

	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
	"github.com/rs/zerolog"
)

type FetchState struct {
	sync.Mutex
	// latestFetchID is relevant later on to return the latest run's details.
	latestFetchID moltservice.FetchAttemptID
	idToRun       map[moltservice.FetchAttemptID]FetchDetail
	// orderedIdList gives the ordered list of fetch attempts.
	orderedIdList []moltservice.FetchAttemptID
}

type VerifyState struct {
	sync.Mutex
	// latestFetchID is relevant later on to return the latest run's details.
	latestVerifyID moltservice.VerifyAttemptID
	idToRun        map[moltservice.VerifyAttemptID]VerifyDetail
	// orderedIdList gives the ordered list of fetch attempts.
	orderedIdList []moltservice.VerifyAttemptID
}

var _ moltservice.Service = &moltService{}

type moltService struct {
	logger       zerolog.Logger
	debugEnabled bool
	fetchState   *FetchState
	verifyState  *VerifyState
}

const MOLTCommand = "./molt"

var fetch = "fetch"

const verify = "verify"

func NewMOLTService(cfg *ServerConfig) (*moltService, error) {
	svc := &moltService{
		logger:       cfg.Logger,
		debugEnabled: cfg.DebugMode,
		fetchState: &FetchState{
			idToRun:       make(map[moltservice.FetchAttemptID]FetchDetail),
			latestFetchID: 0,
			orderedIdList: make([]moltservice.FetchAttemptID, 0),
		},
		verifyState: &VerifyState{
			idToRun:        make(map[moltservice.VerifyAttemptID]VerifyDetail),
			latestVerifyID: 0,
			orderedIdList:  make([]moltservice.VerifyAttemptID, 0),
		},
	}
	return svc, nil
}
