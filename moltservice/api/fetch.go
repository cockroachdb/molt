package api

import (
	"context"
	"fmt"

	"github.com/cockroachdb/molt/moltservice/gen/moltservice"
)

func (m *moltService) CreateFetchTask(
	ctx context.Context, payload *moltservice.CreateFetchPayload,
) (res moltservice.FetchAttemptID, err error) {
	fmt.Printf("%#v", payload)
	return 1, nil
}
