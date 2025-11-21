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
	"github.com/cloudwego/eino/schema"
)

// AgentInput is the input for an agent
type AgentInput struct {
	Messages []*schema.Message
}

// AgentOutput is the output from an agent
type AgentOutput struct {
	Message *schema.Message
}

// AgentAction represents actions that can be taken by an agent
type AgentAction struct {
	Exit            bool
	Interrupted     bool
	TransferToAgent string
	BreakLoop       bool
}

// InterruptInfo contains information about an interrupt
type InterruptInfo struct {
	CheckpointID string
	Info         any
}

// ToolCall represents a tool call from LLM
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// AgentEvent is an event emitted during agent execution
type AgentEvent struct {
	AgentName     string
	RunPath       []string
	Type          string // "llm_start", "llm_response", "tool_start", "tool_result", "completed", "error", "interrupted"
	Output        *AgentOutput
	Action        *AgentAction
	ToolCall      *ToolCall
	ToolResult    string
	InterruptInfo *InterruptInfo
	Err           error
}

// ResumeInput is the input for resuming an interrupted agent
type ResumeInput struct {
	Message *schema.Message
	Data    any
}

// WorkflowParams is the params for temporal workflow
type WorkflowParams struct {
	Messages       []*schema.Message
	RunPath        []string
	RootWorkflowID string // for event routing
}

// WorkflowResult is the result from temporal workflow
type WorkflowResult struct {
	Output        string
	Action        *AgentAction
	InterruptInfo *InterruptInfo
	Events        []*AgentEvent
}

// LLMRequest is the request to LLM activity
type LLMRequest struct {
	Messages    []*schema.Message
	Instruction string
	Tools       []ToolInfo
}

// LLMResponse is the response from LLM activity
type LLMResponse struct {
	Content   string
	ToolCalls []ToolCall
}

// ToolInfo contains tool metadata for LLM
type ToolInfo struct {
	Name   string
	Desc   string
	Schema any
}
