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
)

// SequentialAgentConfig is the configuration for SequentialAgent
type SequentialAgentConfig struct {
	Name        string
	Description string
	SubAgents   []Agent
}

// SequentialAgent executes sub-agents sequentially
type SequentialAgent struct {
	name        string
	description string
	subAgents   []Agent
}

// NewSequentialAgent creates a new SequentialAgent
func NewSequentialAgent(ctx context.Context, config *SequentialAgentConfig) (Agent, error) {
	return &SequentialAgent{
		name:        config.Name,
		description: config.Description,
		subAgents:   config.SubAgents,
	}, nil
}

func (a *SequentialAgent) Name(ctx context.Context) string {
	return a.name
}

func (a *SequentialAgent) Description(ctx context.Context) string {
	return a.description
}

func (a *SequentialAgent) SubAgents() []Agent {
	return a.subAgents
}

// ParallelAgentConfig is the configuration for ParallelAgent
type ParallelAgentConfig struct {
	Name        string
	Description string
	SubAgents   []Agent
}

// ParallelAgent executes sub-agents in parallel
type ParallelAgent struct {
	name        string
	description string
	subAgents   []Agent
}

// NewParallelAgent creates a new ParallelAgent
func NewParallelAgent(ctx context.Context, config *ParallelAgentConfig) (Agent, error) {
	return &ParallelAgent{
		name:        config.Name,
		description: config.Description,
		subAgents:   config.SubAgents,
	}, nil
}

func (a *ParallelAgent) Name(ctx context.Context) string {
	return a.name
}

func (a *ParallelAgent) Description(ctx context.Context) string {
	return a.description
}

func (a *ParallelAgent) SubAgents() []Agent {
	return a.subAgents
}

// LoopAgentConfig is the configuration for LoopAgent
type LoopAgentConfig struct {
	Name         string
	Description  string
	SubAgents    []Agent
	MaxIteration int
}

// LoopAgent executes sub-agents in a loop
type LoopAgent struct {
	name         string
	description  string
	subAgents    []Agent
	maxIteration int
}

// NewLoopAgent creates a new LoopAgent
func NewLoopAgent(ctx context.Context, config *LoopAgentConfig) (Agent, error) {
	maxIter := config.MaxIteration
	if maxIter == 0 {
		maxIter = 10
	}

	return &LoopAgent{
		name:         config.Name,
		description:  config.Description,
		subAgents:    config.SubAgents,
		maxIteration: maxIter,
	}, nil
}

func (a *LoopAgent) Name(ctx context.Context) string {
	return a.name
}

func (a *LoopAgent) Description(ctx context.Context) string {
	return a.description
}

func (a *LoopAgent) SubAgents() []Agent {
	return a.subAgents
}

func (a *LoopAgent) GetMaxIteration() int {
	return a.maxIteration
}
