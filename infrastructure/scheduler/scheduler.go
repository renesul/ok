package scheduler

import (
	"context"
	"time"

	"github.com/renesul/ok/application"
	"go.uber.org/zap"
)

const (
	tickInterval  = 10 * time.Second
	jobTimeout    = 30 * time.Second
	maxFailCount  = 3
)

type Scheduler struct {
	repo         *JobRepository
	agentService *application.AgentService
	log          *zap.Logger
	stopCh       chan struct{}
}

func NewScheduler(repo *JobRepository, agentService *application.AgentService, log *zap.Logger) *Scheduler {
	return &Scheduler{
		repo:         repo,
		agentService: agentService,
		log:          log.Named("scheduler"),
		stopCh:       make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.log.Debug("scheduler started")
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stopCh:
			s.log.Debug("scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) tick() {
	ctx := context.Background()
	jobs, err := s.repo.FindEnabled(ctx)
	if err != nil {
		s.log.Debug("scheduler tick error", zap.Error(err))
		return
	}

	for _, job := range jobs {
		interval := time.Duration(job.IntervalSeconds) * time.Second
		if time.Since(job.LastRun) >= interval {
			go s.runJobByID(job.ID)
		}
	}
}

func (s *Scheduler) runJobByID(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	job, err := s.repo.FindByID(ctx, id)
	if err != nil || job == nil {
		return
	}

	s.log.Debug("job started", zap.String("id", job.ID), zap.String("name", job.Name))

	start := time.Now()
	_, runErr := s.agentService.Run(ctx, job.Input)
	elapsed := time.Since(start)

	if runErr != nil {
		job.FailCount++
		enabled := job.FailCount < maxFailCount
		s.repo.UpdateRun(ctx, job.ID, "failed", job.FailCount, enabled)
		s.log.Debug("job failed",
			zap.String("id", job.ID),
			zap.String("name", job.Name),
			zap.Error(runErr),
			zap.Int("fail_count", job.FailCount),
			zap.Bool("still_enabled", enabled),
			zap.Duration("elapsed", elapsed),
		)
		return
	}

	s.repo.UpdateRun(ctx, job.ID, "ok", 0, true)
	s.log.Debug("job done",
		zap.String("id", job.ID),
		zap.String("name", job.Name),
		zap.Duration("elapsed", elapsed),
	)
}
