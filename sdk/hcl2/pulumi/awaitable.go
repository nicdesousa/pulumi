package pulumi

import "sync"

const (
	awaitablePending  = 0
	awaitableResolved = 1
	awaitableRejected = 2
	awaitableCanceled = 3
)

type awaitable struct {
	mutex sync.Mutex
	cond  *sync.Cond

	state uint32
}

func newAwaitable() *awaitable {
	a := &awaitable{}
	a.cond = sync.NewCond(&a.mutex)
	return a
}

func (a *awaitable) fulfill(state uint32) {
	a.mutex.Lock()
	a.state = state
	a.mutex.Unlock()
	a.cond.Broadcast()
}

func (a *awaitable) await(ctx *programContext) bool {
	a.mutex.Lock()
	for a.state == awaitablePending {
		if ctx.cancel.Err() != nil {
			return false
		}
		a.cond.Wait()
	}
	a.mutex.Unlock()

	return a.state == awaitableResolved

}
