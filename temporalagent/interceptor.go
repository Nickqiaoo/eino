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
	"encoding/json"
	"strings"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"

	"github.com/cloudwego/eino/schema"
)

// AgentInterceptor implements Temporal WorkerInterceptor for agent callbacks
type AgentInterceptor struct {
	interceptor.WorkerInterceptorBase
	registry *CallbackRegistry
}

// NewAgentInterceptor creates a new agent interceptor
func NewAgentInterceptor() *AgentInterceptor {
	return &AgentInterceptor{
		registry: GetCallbackRegistry(),
	}
}

// InterceptWorkflow intercepts workflow execution
func (i *AgentInterceptor) InterceptWorkflow(
	ctx workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &workflowInterceptor{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		registry:                       i.registry,
	}
}

// InterceptActivity intercepts activity execution
func (i *AgentInterceptor) InterceptActivity(
	ctx context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &activityInterceptor{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		registry:                       i.registry,
	}
}

// workflowInterceptor handles workflow-level callbacks
type workflowInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
	registry *CallbackRegistry
}

// ExecuteWorkflow intercepts workflow execution for BeforeAgent/AfterAgent callbacks
func (w *workflowInterceptor) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (interface{}, error) {
	info := workflow.GetInfo(ctx)
	agentName := info.WorkflowType.Name
	callbacks := w.registry.Get(agentName)

	cbCtx := &CallbackContext{
		AgentName:    agentName,
		WorkflowID:   info.WorkflowExecution.ID,
		InvocationID: info.WorkflowExecution.RunID,
	}

	// BeforeAgent callback (executed via SideEffect for determinism)
	if callbacks != nil && callbacks.BeforeAgent != nil {
		// Note: BeforeAgent is handled in workflow code, not interceptor
		// because we need to modify the input which requires workflow context
	}

	// Execute workflow
	result, err := w.Next.ExecuteWorkflow(ctx, in)

	// AfterAgent callback
	if callbacks != nil && callbacks.AfterAgent != nil {
		if wfResult, ok := result.(*WorkflowResult); ok {
			// Execute via SideEffect
			var modifiedResult *WorkflowResult
			workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
				modified, modErr := callbacks.AfterAgent(context.Background(), cbCtx, wfResult, err)
				if modErr != nil {
					return wfResult
				}
				return modified
			}).Get(&modifiedResult)
			if modifiedResult != nil {
				return modifiedResult, err
			}
		}
	}

	return result, err
}

// activityInterceptor handles activity-level callbacks
type activityInterceptor struct {
	interceptor.ActivityInboundInterceptorBase
	registry *CallbackRegistry
}

// ExecuteActivity intercepts activity execution for BeforeModel/AfterModel and BeforeTool/AfterTool callbacks
func (a *activityInterceptor) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (interface{}, error) {
	info := activity.GetInfo(ctx)
	activityName := info.ActivityType.Name

	// Extract agent name from workflow ID (format: agent-{name}-{timestamp}/...)
	agentName := extractAgentNameFromWorkflowID(info.WorkflowExecution.ID)
	callbacks := a.registry.Get(agentName)

	if callbacks == nil {
		return a.Next.ExecuteActivity(ctx, in)
	}

	cbCtx := &CallbackContext{
		AgentName:    agentName,
		WorkflowID:   info.WorkflowExecution.ID,
		InvocationID: info.WorkflowExecution.RunID,
	}

	// Handle LLM call
	if activityName == "llm_call" {
		return a.executeLLMWithCallbacks(ctx, in, cbCtx, callbacks)
	}

	// Handle Tool call
	return a.executeToolWithCallbacks(ctx, in, cbCtx, callbacks, activityName)
}

func (a *activityInterceptor) executeLLMWithCallbacks(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
	cbCtx *CallbackContext,
	callbacks *CallbackHandler,
) (interface{}, error) {
	// Get request from args
	req, ok := in.Args[0].(LLMRequest)
	if !ok {
		return a.Next.ExecuteActivity(ctx, in)
	}

	// BeforeModel callback
	if callbacks.BeforeModel != nil {
		resp, err := callbacks.BeforeModel(ctx, cbCtx, &req)
		if err != nil {
			return nil, err
		}
		if resp != nil {
			// Skip LLM call, return callback response
			return resp, nil
		}
	}

	// Execute LLM activity
	result, err := a.Next.ExecuteActivity(ctx, in)

	// AfterModel callback
	if callbacks.AfterModel != nil {
		var resp *LLMResponse
		if result != nil {
			if r, ok := result.(*LLMResponse); ok {
				resp = r
			}
		}
		modifiedResp, modErr := callbacks.AfterModel(ctx, cbCtx, resp, err)
		if modErr != nil {
			return nil, modErr
		}
		if modifiedResp != nil {
			return modifiedResp, nil
		}
	}

	return result, err
}

func (a *activityInterceptor) executeToolWithCallbacks(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
	cbCtx *CallbackContext,
	callbacks *CallbackHandler,
	toolName string,
) (interface{}, error) {
	// Get args from input
	argsStr, ok := in.Args[0].(string)
	if !ok {
		return a.Next.ExecuteActivity(ctx, in)
	}

	// Parse args to map
	var args map[string]any
	json.Unmarshal([]byte(argsStr), &args)

	// Create tool info
	toolInfo := &schema.ToolInfo{
		Name: toolName,
	}

	// BeforeTool callback
	if callbacks.BeforeTool != nil {
		modifiedArgs, err := callbacks.BeforeTool(ctx, cbCtx, toolInfo, args)
		if err != nil {
			return nil, err
		}
		if modifiedArgs != nil {
			// Use modified args
			modifiedArgsBytes, _ := json.Marshal(modifiedArgs)
			in.Args[0] = string(modifiedArgsBytes)
		}
	}

	// Execute tool activity
	result, err := a.Next.ExecuteActivity(ctx, in)

	// AfterTool callback
	if callbacks.AfterTool != nil {
		var resultMap map[string]any
		if resultStr, ok := result.(string); ok {
			json.Unmarshal([]byte(resultStr), &resultMap)
			if resultMap == nil {
				resultMap = map[string]any{"result": resultStr}
			}
		}

		modifiedResult, modErr := callbacks.AfterTool(ctx, cbCtx, toolInfo, args, resultMap, err)
		if modErr != nil {
			return nil, modErr
		}
		if modifiedResult != nil {
			modifiedResultBytes, _ := json.Marshal(modifiedResult)
			return string(modifiedResultBytes), nil
		}
	}

	return result, err
}

// extractAgentNameFromWorkflowID extracts agent name from workflow ID
// Format: agent-{name}-{timestamp} or agent-{name}-{timestamp}/subworkflow
func extractAgentNameFromWorkflowID(workflowID string) string {
	// Remove any subworkflow paths
	parts := strings.Split(workflowID, "/")
	mainID := parts[0]

	// Parse agent-{name}-{timestamp}
	if strings.HasPrefix(mainID, "agent-") {
		remaining := strings.TrimPrefix(mainID, "agent-")
		// Find last dash (before timestamp)
		lastDash := strings.LastIndex(remaining, "-")
		if lastDash > 0 {
			return remaining[:lastDash]
		}
	}

	return mainID
}
