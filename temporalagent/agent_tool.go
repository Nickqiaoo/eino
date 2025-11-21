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

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/temporalagent/tool"
)

// AgentTool wraps an Agent as a Tool
type AgentTool struct {
	agent Agent
}

// NewAgentTool creates a new AgentTool from an Agent
func NewAgentTool(ctx context.Context, agent Agent) tool.InvokableTool {
	return &AgentTool{agent: agent}
}

func (t *AgentTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params := map[string]*schema.ParameterInfo{
		"message": {
			Type:     schema.String,
			Desc:     "The message to send to the agent",
			Required: true,
		},
	}

	return &schema.ToolInfo{
		Name:        "agent_" + t.agent.Name(ctx),
		Desc:        t.agent.Description(ctx),
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

// InvokableRun should not be called directly - handled by workflow
func (t *AgentTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	panic("AgentTool.InvokableRun should not be called directly - it is handled by workflow as child workflow")
}

// GetAgent returns the wrapped agent
func (t *AgentTool) GetAgent() Agent {
	return t.agent
}

// IsAgentTool checks if a tool is an AgentTool
func IsAgentTool(t tool.InvokableTool) bool {
	_, ok := t.(*AgentTool)
	return ok
}
