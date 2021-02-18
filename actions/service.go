package actions

//go:generate mockgen -package=mock -destination=mock/mock_actions.go github.com/treeverse/lakefs/actions Source,OutputWriter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/treeverse/lakefs/db"
)

type Service struct {
	DB db.Database
}

type Task struct {
	RunID     string
	HookRunID string
	Action    *Action
	HookID    string
	Hook      Hook
	Err       error
	StartTime time.Time
	EndTime   time.Time
}

type RunResult struct {
	RunID     string    `db:"run_id"`
	BranchID  string    `db:"branch_id"`
	SourceRef string    `db:"source_ref"`
	EventType string    `db:"event_type"`
	StartTime time.Time `db:"start_time"`
	EndTime   time.Time `db:"end_time"`
	Passed    bool      `db:"passed"`
	CommitID  string    `db:"commit_id"`
}

type TaskResult struct {
	RunID      string    `db:"run_id"`
	HookRunID  string    `db:"hook_run_id"`
	HookID     string    `db:"hook_id"`
	ActionName string    `db:"action_name"`
	StartTime  time.Time `db:"start_time"`
	EndTime    time.Time `db:"end_time"`
	Passed     bool      `db:"passed"`
}

func (r *TaskResult) LogPath() string {
	return FormatHookOutputPath(r.RunID, r.ActionName, r.HookID)
}

type RunResultIterator interface {
	Next() bool
	Value() *RunResult
	SeekGE(runID string)
	Err() error
	Close()
}

type TaskResultIterator interface {
	Next() bool
	Value() *TaskResult
	Err() error
	Close()
}

const defaultFetchSize = 1024

var ErrNotFound = errors.New("not found")

func NewService(db db.Database) *Service {
	return &Service{
		DB: db,
	}
}

// Run load and run actions based on the event information. Always returns run id in order to track back hooks errors
//
func (s *Service) Run(ctx context.Context, event Event, deps Deps) (string, error) {
	runID := NewRunID()

	// load relevant actions
	actions, err := s.loadMatchedActions(ctx, deps.Source, MatchSpec{EventType: event.Type, Branch: event.BranchID})
	if err != nil || len(actions) == 0 {
		return runID, err
	}

	// allocate and run hooks
	tasks, err := s.allocateTasks(runID, actions)
	if err != nil {
		return runID, err
	}

	runErr := s.runTasks(ctx, tasks, event, deps)
	// keep results before returning an error (if any)
	err = s.insertRunInformation(ctx, runID, event, tasks, runErr)
	if err != nil {
		return runID, err
	}
	return runID, runErr
}

func (s *Service) loadMatchedActions(ctx context.Context, source Source, spec MatchSpec) ([]*Action, error) {
	if source == nil {
		return nil, nil
	}
	actions, err := LoadActions(ctx, source)
	if err != nil {
		return nil, err
	}
	return MatchedActions(actions, spec)
}

func (s *Service) allocateTasks(runID string, actions []*Action) ([]*Task, error) {
	var tasks []*Task
	for _, action := range actions {
		for _, hook := range action.Hooks {
			h, err := NewHook(hook, action)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, &Task{
				RunID:     runID,
				HookRunID: NewRunID(),
				Action:    action,
				HookID:    hook.ID,
				Hook:      h,
			})
		}
	}
	return tasks, nil
}

func (s *Service) runTasks(ctx context.Context, tasks []*Task, event Event, deps Deps) error {
	var g multierror.Group
	for _, task := range tasks {
		task := task // pin
		g.Go(func() error {
			task.StartTime = time.Now()
			err := task.Hook.Run(ctx, event, &HookOutputWriter{
				RunID:      task.RunID,
				HookRunID:  task.HookRunID,
				ActionName: task.Action.Name,
				HookID:     task.HookID,
				Writer:     deps.Output,
			})
			if err != nil {
				task.Err = fmt.Errorf("run '%s' failed on action '%s' hook '%s': %w",
					task.RunID, task.Action.Name, task.HookID, err)
			}
			task.EndTime = time.Now()
			return task.Err
		})
	}
	return g.Wait().ErrorOrNil()
}

func (s *Service) insertRunInformation(ctx context.Context, runID string, event Event, tasks []*Task, runErr error) error {
	if len(tasks) == 0 {
		return nil
	}
	_, err := s.DB.Transact(func(tx db.Tx) (interface{}, error) {
		runEndTime := getMaxEndTime(tasks)

		// insert run information
		runPassed := runErr == nil
		_, err := tx.Exec(`INSERT INTO actions_runs(repository_id, run_id, event_type, start_time, end_time, branch_id, source_ref, commit_id, passed)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'',$8)`,
			event.RepositoryID, runID, event.Type, event.Time, runEndTime, event.BranchID, event.SourceRef, runPassed)
		if err != nil {
			return nil, fmt.Errorf("insert run information: %w", err)
		}

		// insert each task information
		for _, task := range tasks {
			taskPassed := task.Err == nil
			_, err = tx.Exec(`INSERT INTO actions_run_hooks(repository_id, run_id, hook_run_id, event_type, action_name, hook_id, start_time, end_time, passed)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				event.RepositoryID, runID, task.HookRunID, event.Type, task.Action.Name, task.HookID, task.StartTime, task.EndTime, taskPassed)
			if err != nil {
				return nil, fmt.Errorf("insert run hook information (%s %s): %w", task.Action.Name, task.HookID, err)
			}
		}
		return nil, nil
	}, db.WithContext(ctx))
	return err
}

func getMaxEndTime(tasks []*Task) time.Time {
	var endTime time.Time
	for _, task := range tasks {
		if task.EndTime.After(endTime) {
			endTime = task.EndTime
		}
	}
	return endTime
}

func (s *Service) UpdateCommitID(ctx context.Context, repositoryID string, runID string, commitID string) error {
	_, err := s.DB.Transact(func(tx db.Tx) (interface{}, error) {
		_, err := tx.Exec(`UPDATE actions_runs SET commit_id=$3 WHERE repository_id=$1 AND run_id=$2`,
			repositoryID, runID, commitID)
		if err != nil {
			return nil, fmt.Errorf("update run commit_id: %w", err)
		}
		return nil, nil
	}, db.WithContext(ctx))
	return err
}

func (s *Service) GetRunResult(ctx context.Context, repositoryID string, runID string) (*RunResult, error) {
	res, err := s.DB.Transact(func(tx db.Tx) (interface{}, error) {
		result := &RunResult{
			RunID: runID,
		}
		err := tx.Get(result, `SELECT event_type, branch_id, source_ref, start_time, end_time, passed, commit_id
			FROM actions_runs
			WHERE repository_id=$1 AND run_id=$2`,
			repositoryID, runID)
		if err != nil {
			return nil, fmt.Errorf("get run result: %w", err)
		}
		return result, nil
	}, db.WithContext(ctx), db.ReadOnly())
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return res.(*RunResult), nil
}

func (s *Service) GetTaskResult(ctx context.Context, repositoryID string, runID string, hookRunID string) (*TaskResult, error) {
	res, err := s.DB.Transact(func(tx db.Tx) (interface{}, error) {
		result := &TaskResult{
			RunID:     runID,
			HookRunID: hookRunID,
		}
		err := tx.Get(result, `SELECT hook_id, action_name, start_time, end_time, passed
			FROM actions_run_hooks 
			WHERE repository_id=$1 AND run_id=$2 AND hook_run_id=$3`,
			repositoryID, runID, hookRunID)
		if err != nil {
			return nil, fmt.Errorf("get task result: %w", err)
		}
		return result, nil
	}, db.WithContext(ctx), db.ReadOnly())
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return res.(*TaskResult), nil
}

func (s *Service) ListRuns(ctx context.Context, repositoryID string, branchID *string, after string) (RunResultIterator, error) {
	iter := NewDBRunResultIterator(ctx, s.DB, defaultFetchSize, repositoryID, branchID, after)
	return iter, nil
}

func (s *Service) ListRunTasks(ctx context.Context, repositoryID string, runID string, after string) (TaskResultIterator, error) {
	iter := NewDBTaskResultIterator(ctx, s.DB, defaultFetchSize, repositoryID, runID, after)
	return iter, nil
}
