package api

import (
	"fmt"

	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
	"github.com/rs/zerolog"
)

var _ moltservice.Service = &moltService{}

type moltService struct {
	Logger zerolog.Logger
}

const MOLTCommand = "./molt"

var MOLTFetchCommand = fmt.Sprintf("%s fetch", MOLTCommand)

func NewMOLTService(cfg *ServerConfig) (*moltService, error) {
	svc := &moltService{
		Logger: cfg.Logger,
	}
	return svc, nil
}
