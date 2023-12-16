package rowverify

import (
	"container/heap"

	"github.com/cockroachdb/molt/verify/verifymetrics"
)

// liveRetryQueue implements the heap interface.
type liveRetryQueue struct {
	items  []*liveRetryItem
	numPKs int
}

var _ heap.Interface = (*liveRetryQueue)(nil)

func (rq *liveRetryQueue) heapPush(it *liveRetryItem) {
	heap.Push(rq, it)
}

func (rq *liveRetryQueue) heapPop() *liveRetryItem {
	return heap.Pop(rq).(*liveRetryItem)
}

func (rq liveRetryQueue) Len() int { return len(rq.items) }

func (rq liveRetryQueue) Less(i, j int) bool {
	return rq.items[i].Retry.NextRetry.Before(rq.items[j].Retry.NextRetry)
}

func (rq liveRetryQueue) Swap(i, j int) {
	rq.items[i], rq.items[j] = rq.items[j], rq.items[i]
}

func (rq *liveRetryQueue) Push(x any) {
	rq.items = append(rq.items, x.(*liveRetryItem))
	rq.numPKs += len(x.(*liveRetryItem).PrimaryKeys)
	verifymetrics.LivePrimaryKeys.Set(float64(rq.numPKs))
	verifymetrics.LiveBatches.Set(float64(len(rq.items)))
}

func (rq *liveRetryQueue) Pop() any {
	old := rq.items
	n := len(old)
	item := old[n-1]
	rq.items = old[0 : n-1]
	rq.numPKs -= len(item.PrimaryKeys)
	verifymetrics.LivePrimaryKeys.Set(float64(rq.numPKs))
	verifymetrics.LiveBatches.Set(float64(len(rq.items)))
	return item
}
