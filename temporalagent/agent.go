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

	"github.com/cloudwego/eino/temporalagent/tool"
)

// AgentRunOption is the option for running an agent
type AgentRunOption func(*AgentRunOptions)

// AgentRunOptions contains options for running an agent
type AgentRunOptions struct {
	// Add options as needed
}

// Agent is the interface for all agents
type Agent interface {
	Name(ctx context.Context) string
	Description(ctx context.Context) string
}

// SubAgentHolder is an interface for agents that hold sub-agents
type SubAgentHolder interface {
	SubAgents() []Agent
}

// ToolHolder is an interface for agents that hold tools
type ToolHolder interface {
	Tools() []tool.InvokableTool
}

// ToolsConfig contains tool configuration for an agent
type ToolsConfig struct {
	Tools []tool.InvokableTool
}
