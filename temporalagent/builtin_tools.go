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

// HumanInputToolName is the name of the human input tool
const HumanInputToolName = "human_input"

// humanInputTool is a built-in tool that triggers an interrupt for human input
type humanInputTool struct{}

// NewHumanInputTool creates a new human input tool
func NewHumanInputTool() tool.InvokableTool {
	return &humanInputTool{}
}

func (t *humanInputTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params := map[string]*schema.ParameterInfo{
		"question": {
			Type:     schema.String,
			Desc:     "The question to ask the human",
			Required: true,
		},
	}

	return &schema.ToolInfo{
		Name:        HumanInputToolName,
		Desc:        "Ask a human for input. Use this when you need clarification or additional information from the user.",
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

// InvokableRun is handled specially by the workflow - it triggers an interrupt
func (t *humanInputTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// This should not be called directly - handled by workflow
	panic("human_input tool should be handled by workflow, not called directly")
}

// IsHumanInputTool checks if a tool call is for human input
func IsHumanInputTool(toolName string) bool {
	return toolName == HumanInputToolName
}

// ExitToolName is the name of the exit tool
const ExitToolName = "exit"

// exitTool is a built-in tool that exits the agent loop
type exitTool struct{}

// NewExitTool creates a new exit tool
func NewExitTool() tool.InvokableTool {
	return &exitTool{}
}

func (t *exitTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	params := map[string]*schema.ParameterInfo{
		"result": {
			Type:     schema.String,
			Desc:     "The final result to return",
			Required: true,
		},
	}

	return &schema.ToolInfo{
		Name:        ExitToolName,
		Desc:        "Exit the agent and return a final result. Use this when you have completed the task.",
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

func (t *exitTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// This should not be called directly - handled by workflow
	panic("exit tool should be handled by workflow, not called directly")
}

// IsExitTool checks if a tool call is for exit
func IsExitTool(toolName string) bool {
	return toolName == ExitToolName
}
