package watchlist

import (
	"sync"
	"time"
)

// JobState is the lifecycle of a background job.
type JobState string

const (
	JobRunning JobState = "running"
	JobDone    JobState = "done"
	JobFailed  JobState = "failed"
)

// JobUpdate is one SSE frame / status snapshot.
type JobUpdate struct {
	ID       string   `json:"id"`
	State    JobState `json:"state"`
	Phase    string   `json:"phase"`
	Current  int      `json:"current"`
	Total    int      `json:"total"`
	Message  string   `json:"message,omitempty"`
	Result   any      `json:"result,omitempty"`
	Error    *JobErr  `json:"error,omitempty"`
	Finished bool     `json:"finished"`
}

// JobErr is a classified failure with optional guidance.
type JobErr struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// Job is a unit of long-running work with progress subscribers.
type Job struct {
	ID        string
	CreatedAt time.Time

	mu      sync.Mutex
	last    JobUpdate
	subs    map[chan JobUpdate]struct{}
	closed  bool
	retain  time.Duration
	expires time.Time
}

// Jobs is a registry of running and recently finished jobs.
type Jobs struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

// NewJobs creates a registry and starts its reaper.
func NewJobs() *Jobs {
	j := &Jobs{jobs: map[string]*Job{}}
	go j.reap()
	return j
}

// Create registers a new running job.
func (js *Jobs) Create(phase string) *Job {
	job := &Job{
		ID:        NewID(),
		CreatedAt: time.Now(),
		subs:      map[chan JobUpdate]struct{}{},
		retain:    10 * time.Minute,
	}
	job.last = JobUpdate{ID: job.ID, State: JobRunning, Phase: phase}

	js.mu.Lock()
	js.jobs[job.ID] = job
	js.mu.Unlock()
	return job
}

// Get returns a job by id.
func (js *Jobs) Get(id string) (*Job, bool) {
	js.mu.Lock()
	defer js.mu.Unlock()
	j, ok := js.jobs[id]
	return j, ok
}

func (js *Jobs) reap() {
	for range time.Tick(time.Minute) {
		now := time.Now()
		js.mu.Lock()
		for id, j := range js.jobs {
			j.mu.Lock()
			expired := j.closed && !j.expires.IsZero() && now.After(j.expires)
			j.mu.Unlock()
			if expired {
				delete(js.jobs, id)
			}
		}
		js.mu.Unlock()
	}
}

// Subscribe returns a channel of updates, pre-loaded with the current state.
func (j *Job) Subscribe() (<-chan JobUpdate, func()) {
	ch := make(chan JobUpdate, 16)

	j.mu.Lock()
	current := j.last
	closed := j.closed
	if !closed {
		j.subs[ch] = struct{}{}
	}
	j.mu.Unlock()

	ch <- current
	if closed {
		close(ch)
		return ch, func() {}
	}

	return ch, func() {
		j.mu.Lock()
		if _, ok := j.subs[ch]; ok {
			delete(j.subs, ch)
			close(ch)
		}
		j.mu.Unlock()
	}
}

func (j *Job) publish(u JobUpdate) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return
	}
	j.last = u
	for ch := range j.subs {
		select {
		case ch <- u:
		default:
		}
	}
	if u.Finished {
		j.closed = true
		j.expires = time.Now().Add(j.retain)
		for ch := range j.subs {
			close(ch)
			delete(j.subs, ch)
		}
	}
}

// Progress reports incremental progress.
func (j *Job) Progress(phase string, current, total int) {
	j.mu.Lock()
	u := j.last
	j.mu.Unlock()
	u.State, u.Phase, u.Current, u.Total = JobRunning, phase, current, total
	j.publish(u)
}

// Message sets a status line without changing the counters.
func (j *Job) Message(msg string) {
	j.mu.Lock()
	u := j.last
	j.mu.Unlock()
	u.Message = msg
	j.publish(u)
}

// Done finishes the job successfully.
func (j *Job) Done(result any) {
	j.mu.Lock()
	u := j.last
	j.mu.Unlock()
	u.State, u.Result, u.Finished, u.Error = JobDone, result, true, nil
	j.publish(u)
}

// Fail finishes the job with a classified error.
func (j *Job) Fail(kind, msg, hint string) {
	j.mu.Lock()
	u := j.last
	j.mu.Unlock()
	u.State, u.Finished = JobFailed, true
	u.Error = &JobErr{Kind: kind, Message: msg, Hint: hint}
	j.publish(u)
}

// Snapshot returns the latest update.
func (j *Job) Snapshot() JobUpdate {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.last
}
