/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package temporalagent

import (
	"context"
	"time"
)

// RunnerConfig is the configuration for Runner
type RunnerConfig struct {
	TemporalHost string // default: localhost:7233
	TaskQueue    string // default: agent-tasks

	// Timeout options
	LLMTimeout  time.Duration // default: 2 minutes
	ToolTimeout time.Duration // default: 5 minutes
}

// Runner manages agent execution
type Runner struct {
	agent    Agent
	executor *TemporalExecutor
}

// NewRunner creates a new Runner
func NewRunner(ctx context.Context, agent Agent, config *RunnerConfig) (*Runner, error) {
	if config == nil {
		config = &RunnerConfig{}
	}

	executor, err := NewTemporalExecutor(&ExecutorConfig{
		TemporalHost: config.TemporalHost,
		TaskQueue:    config.TaskQueue,
		LLMTimeout:   config.LLMTimeout,
		ToolTimeout:  config.ToolTimeout,
	})
	if err != nil {
		return nil, err
	}

	// Register the entire agent tree
	executor.RegisterAgentTree(agent)

	// Start the worker
	if err := executor.Start(); err != nil {
		return nil, err
	}

	return &Runner{
		agent:    agent,
		executor: executor,
	}, nil
}

// Run starts the agent execution and returns a channel of events
func (r *Runner) Run(ctx context.Context, input *AgentInput) <-chan *AgentEvent {
	agentName := r.agent.Name(ctx)

	workflowID, err := r.executor.StartWorkflow(ctx, agentName, input)
	if err != nil {
		ch := make(chan *AgentEvent, 1)
		ch <- &AgentEvent{Err: err}
		close(ch)
		return ch
	}

	return r.executor.SubscribeEvents(ctx, workflowID)
}

// Resume resumes an interrupted agent
func (r *Runner) Resume(ctx context.Context, checkpointID string, input *ResumeInput) <-chan *AgentEvent {
	err := r.executor.Resume(ctx, checkpointID, input)
	if err != nil {
		ch := make(chan *AgentEvent, 1)
		ch <- &AgentEvent{Err: err}
		close(ch)
		return ch
	}

	return r.executor.SubscribeEvents(ctx, checkpointID)
}

// Close stops the runner
func (r *Runner) Close() {
	r.executor.Stop()
}
