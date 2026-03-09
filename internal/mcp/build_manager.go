// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Work-Fort/Anvil/pkg/kernel"
)

// BuildJob tracks a single kernel build.
type BuildJob struct {
	ID          string             `json:"build_id"`
	Arch        string             `json:"arch"`
	Version     string             `json:"version"`
	Status      string             `json:"status"` // running, completed, failed, cancelled
	Phase       string             `json:"phase"`
	Progress    float64            `json:"progress"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt time.Time          `json:"completed_at,omitempty"`
	Stats       *kernel.BuildStats `json:"stats,omitempty"`
	Error       string             `json:"error,omitempty"`
	LogLines    []string           `json:"-"`
	cancel      context.CancelFunc
	done        chan struct{}
	mu          sync.RWMutex
}

// BuildManager manages in-flight and completed builds.
type BuildManager struct {
	mu   sync.RWMutex
	jobs map[string]*BuildJob
}

// NewBuildManager creates a new build manager.
func NewBuildManager() *BuildManager {
	return &BuildManager{
		jobs: make(map[string]*BuildJob),
	}
}

// NewJob creates a new build job and registers it.
func (bm *BuildManager) NewJob(version, arch string, cancel context.CancelFunc) *BuildJob {
	id := fmt.Sprintf("b-%d-%s", time.Now().Unix(), arch)
	job := &BuildJob{
		ID:        id,
		Arch:      arch,
		Version:   version,
		Status:    "running",
		Phase:     "starting",
		StartedAt: time.Now(),
		LogLines:  make([]string, 0, 200),
		cancel:    cancel,
		done:      make(chan struct{}),
	}

	bm.mu.Lock()
	bm.jobs[id] = job
	bm.mu.Unlock()

	return job
}

// GetJob returns a job by ID.
func (bm *BuildManager) GetJob(id string) *BuildJob {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.jobs[id]
}

// RunningForArch returns the running job for an architecture, or nil.
func (bm *BuildManager) RunningForArch(arch string) *BuildJob {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	for _, job := range bm.jobs {
		job.mu.RLock()
		status := job.Status
		jobArch := job.Arch
		job.mu.RUnlock()
		if jobArch == arch && status == "running" {
			return job
		}
	}
	return nil
}

// SetPhase updates the build phase.
func (j *BuildJob) SetPhase(phase string) {
	j.mu.Lock()
	j.Phase = phase
	j.Progress = 0
	j.mu.Unlock()
}

// SetProgress updates progress within the current phase.
func (j *BuildJob) SetProgress(pct float64) {
	j.mu.Lock()
	j.Progress = pct
	j.mu.Unlock()
}

// AppendLog adds a line to the log ring buffer.
func (j *BuildJob) AppendLog(line string) {
	j.mu.Lock()
	if len(j.LogLines) >= 200 {
		j.LogLines = j.LogLines[1:]
	}
	j.LogLines = append(j.LogLines, line)
	j.mu.Unlock()
}

// GetLogLines returns the last n log lines.
func (j *BuildJob) GetLogLines(n int) []string {
	j.mu.RLock()
	defer j.mu.RUnlock()
	if n <= 0 || n > len(j.LogLines) {
		n = len(j.LogLines)
	}
	start := len(j.LogLines) - n
	result := make([]string, n)
	copy(result, j.LogLines[start:])
	return result
}

// Complete marks the build as completed.
func (j *BuildJob) Complete(stats *kernel.BuildStats) {
	j.mu.Lock()
	j.Status = "completed"
	j.CompletedAt = time.Now()
	j.Stats = stats
	j.mu.Unlock()
	close(j.done)
}

// Fail marks the build as failed.
func (j *BuildJob) Fail(err error) {
	j.mu.Lock()
	j.Status = "failed"
	j.CompletedAt = time.Now()
	j.Error = err.Error()
	j.mu.Unlock()
	close(j.done)
}

// Cancel cancels the build.
func (j *BuildJob) Cancel() {
	j.mu.Lock()
	if j.Status == "running" {
		j.Status = "cancelled"
		j.CompletedAt = time.Now()
	}
	j.mu.Unlock()
	j.cancel()
	close(j.done)
}

// Wait blocks until the build is done.
func (j *BuildJob) Wait() {
	<-j.done
}

// Snapshot returns a copy of the job state for JSON serialization.
func (j *BuildJob) Snapshot() map[string]any {
	j.mu.RLock()
	defer j.mu.RUnlock()

	result := map[string]any{
		"build_id": j.ID,
		"arch":     j.Arch,
		"version":  j.Version,
		"status":   j.Status,
		"phase":    j.Phase,
		"progress": j.Progress,
		"elapsed":  time.Since(j.StartedAt).Round(time.Second).String(),
	}

	if !j.CompletedAt.IsZero() {
		result["elapsed"] = j.CompletedAt.Sub(j.StartedAt).Round(time.Second).String()
	}

	if j.Stats != nil {
		result["stats"] = j.Stats
	}

	if j.Error != "" {
		result["error"] = j.Error
	}

	return result
}
