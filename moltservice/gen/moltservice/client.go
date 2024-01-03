// Code generated by goa v3.14.1, DO NOT EDIT.
//
// moltservice client
//
// Command:
// $ goa gen github.com/cockroachdb/molt/moltservice/design -o ./moltservice

package moltservice

import (
	"context"

	goa "goa.design/goa/v3/pkg"
)

// Client is the "moltservice" service client.
type Client struct {
	CreateFetchTaskEndpoint      goa.Endpoint
	GetFetchTasksEndpoint        goa.Endpoint
	GetSpecificFetchTaskEndpoint goa.Endpoint
}

// NewClient initializes a "moltservice" service client given the endpoints.
func NewClient(createFetchTask, getFetchTasks, getSpecificFetchTask goa.Endpoint) *Client {
	return &Client{
		CreateFetchTaskEndpoint:      createFetchTask,
		GetFetchTasksEndpoint:        getFetchTasks,
		GetSpecificFetchTaskEndpoint: getSpecificFetchTask,
	}
}

// CreateFetchTask calls the "create_fetch_task" endpoint of the "moltservice"
// service.
func (c *Client) CreateFetchTask(ctx context.Context, p *CreateFetchPayload) (res FetchAttemptID, err error) {
	var ires any
	ires, err = c.CreateFetchTaskEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.(FetchAttemptID), nil
}

// GetFetchTasks calls the "get_fetch_tasks" endpoint of the "moltservice"
// service.
func (c *Client) GetFetchTasks(ctx context.Context) (res []*FetchRun, err error) {
	var ires any
	ires, err = c.GetFetchTasksEndpoint(ctx, nil)
	if err != nil {
		return
	}
	return ires.([]*FetchRun), nil
}

// GetSpecificFetchTask calls the "get_specific_fetch_task" endpoint of the
// "moltservice" service.
func (c *Client) GetSpecificFetchTask(ctx context.Context, p *GetSpecificFetchTaskPayload) (res *FetchRunDetailed, err error) {
	var ires any
	ires, err = c.GetSpecificFetchTaskEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.(*FetchRunDetailed), nil
}