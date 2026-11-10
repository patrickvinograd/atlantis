// Copyright 2025 The Atlantis Authors
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"github.com/runatlantis/atlantis/server/events/command"
	"github.com/runatlantis/atlantis/server/events/models"
	"github.com/runatlantis/atlantis/server/metrics"
)

type IntrumentedCommandRunner interface {
	Plan(ctx command.ProjectContext) command.ProjectResult
	PolicyCheck(ctx command.ProjectContext) command.ProjectResult
	Apply(ctx command.ProjectContext) command.ProjectResult
	ApprovePolicies(ctx command.ProjectContext) command.ProjectResult
	Import(ctx command.ProjectContext) command.ProjectResult
	StateRm(ctx command.ProjectContext) command.ProjectResult
}

type InstrumentedProjectCommandRunner struct {
	projectCommandRunner ProjectCommandRunner
}

func NewInstrumentedProjectCommandRunner(projectCommandRunner ProjectCommandRunner) *InstrumentedProjectCommandRunner {
	return &InstrumentedProjectCommandRunner{
		projectCommandRunner: projectCommandRunner,
	}
}

func (p *InstrumentedProjectCommandRunner) Plan(ctx command.ProjectContext) command.ProjectCommandOutput {
	return RunAndEmitStats(ctx, p.projectCommandRunner.Plan)
}

func (p *InstrumentedProjectCommandRunner) PolicyCheck(ctx command.ProjectContext) command.ProjectCommandOutput {
	return RunAndEmitStats(ctx, p.projectCommandRunner.PolicyCheck)
}

func (p *InstrumentedProjectCommandRunner) Apply(ctx command.ProjectContext) command.ProjectCommandOutput {
	return RunAndEmitStats(ctx, p.projectCommandRunner.Apply)
}

func (p *InstrumentedProjectCommandRunner) PublishDeferredApplyStatuses(projectCmds []command.ProjectContext, result command.Result, status models.CommitStatus) {
	publisher, ok := p.projectCommandRunner.(DeferredApplyStatusPublisher)
	if !ok {
		return
	}
	publisher.PublishDeferredApplyStatuses(projectCmds, result, status)
}

func (p *InstrumentedProjectCommandRunner) ApprovePolicies(ctx command.ProjectContext) command.ProjectCommandOutput {
	return RunAndEmitStats(ctx, p.projectCommandRunner.ApprovePolicies)
}

func (p *InstrumentedProjectCommandRunner) Import(ctx command.ProjectContext) command.ProjectCommandOutput {
	return RunAndEmitStats(ctx, p.projectCommandRunner.Import)
}

func (p *InstrumentedProjectCommandRunner) StateRm(ctx command.ProjectContext) command.ProjectCommandOutput {
	return RunAndEmitStats(ctx, p.projectCommandRunner.StateRm)
}

func RunAndEmitStats(ctx command.ProjectContext, execute func(ctx command.ProjectContext) command.ProjectCommandOutput) command.ProjectCommandOutput {
	commandName := ctx.CommandName.String()
	scope := ctx.Scope
	logger := ctx.Log

	executionTime := scope.Timer(metrics.ExecutionTimeMetric).Start()
	defer executionTime.Stop()

	executionSuccess := scope.Counter(metrics.ExecutionSuccessMetric)
	executionError := scope.Counter(metrics.ExecutionErrorMetric)
	executionFailure := scope.Counter(metrics.ExecutionFailureMetric)

	result := execute(ctx)

	if result.Error != nil {
		executionError.Inc(1)
		logger.Err("Error running %s operation: %s", commandName, result.Error.Error())
		return result
	}

	if result.Failure != "" {
		executionFailure.Inc(1)
		logger.Err("Failure running %s operation: %s", commandName, result.Failure)
		return result
	}

	logger.Info("%s success. output available at: %s", commandName, ctx.Pull.URL)

	executionSuccess.Inc(1)
	return result

}

func (p *InstrumentedProjectCommandRunner) PublishDeferredPlanStatuses(projectCmds []command.ProjectContext, result command.Result, status models.CommitStatus) {
	if publisher, ok := p.projectCommandRunner.(DeferredPlanStatusPublisher); ok {
		publisher.PublishDeferredPlanStatuses(projectCmds, result, status)
	}
}
