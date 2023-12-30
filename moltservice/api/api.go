package api

import "github.com/cockroachdb/molt/moltservice/gen/moltservice"

var _ moltservice.Service = &moltService{}

type moltService struct{}

func NewMOLTService(cfg *ServerConfig) (*moltService, error) {
	svc := &moltService{}
	return svc, nil
}
