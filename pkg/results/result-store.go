package results

import (
	"context"
	"github.com/kluctl/kluctl/v2/pkg/types/result"
)

type ListProjectsOptions struct {
	ProjectFilter *result.ProjectKey `json:"projectFilter,omitempty"`
}

type ListTargetsOptions struct {
	ProjectFilter *result.ProjectKey `json:"projectFilter,omitempty"`
}

type ListCommandResultSummariesOptions struct {
	ProjectFilter *result.ProjectKey `json:"projectFilter,omitempty"`
}

type GetCommandResultOptions struct {
	Id      string `json:"id"`
	Reduced bool   `json:"reduced,omitempty"`
}

type WatchCommandResultSummaryEvent struct {
	Summary *result.CommandResultSummary `json:"summary"`
	Delete  bool                         `json:"delete"`
}

type ResultStore interface {
	WriteCommandResult(cr *result.CommandResult) error

	ListCommandResultSummaries(options ListCommandResultSummariesOptions) ([]result.CommandResultSummary, error)
	WatchCommandResultSummaries(options ListCommandResultSummariesOptions) ([]*result.CommandResultSummary, <-chan WatchCommandResultSummaryEvent, context.CancelFunc, error)
	HasCommandResult(id string) (bool, error)
	GetCommandResultSummary(id string) (*result.CommandResultSummary, error)
	GetCommandResult(options GetCommandResultOptions) (*result.CommandResult, error)
}

func FilterSummary(x *result.CommandResultSummary, filter *result.ProjectKey) bool {
	if filter == nil {
		return true
	}
	if x.ProjectKey.GitRepoKey != filter.GitRepoKey {
		return false
	}
	if x.ProjectKey.SubDir != filter.SubDir {
		return false
	}
	return true
}

func lessSummary(a *result.CommandResultSummary, b *result.CommandResultSummary) bool {
	if a.Command.StartTime != b.Command.StartTime {
		return a.Command.StartTime.After(b.Command.StartTime.Time)
	}
	if a.Command.EndTime != b.Command.EndTime {
		return a.Command.EndTime.After(b.Command.EndTime.Time)
	}
	return a.Command.Command < b.Command.Command
}
