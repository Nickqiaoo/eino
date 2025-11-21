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
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/temporalagent/tool"
)

// ExecutorConfig is the configuration for temporal executor
type ExecutorConfig struct {
	TemporalHost string
	TaskQueue    string

	// Activity options
	LLMTimeout  time.Duration // default: 2 minutes
	ToolTimeout time.Duration // default: 5 minutes
}

// TemporalExecutor manages temporal workflow/activity registration and execution
type TemporalExecutor struct {
	client        client.Client
	worker        worker.Worker
	taskQueue     string
	modelRegistry *ModelRegistry
	llmTimeout    time.Duration
	toolTimeout   time.Duration
}

// NewTemporalExecutor creates a new temporal executor
func NewTemporalExecutor(config *ExecutorConfig) (*TemporalExecutor, error) {
	host := config.TemporalHost
	if host == "" {
		host = "localhost:7233"
	}

	taskQueue := config.TaskQueue
	if taskQueue == "" {
		taskQueue = "agent-tasks"
	}

	c, err := client.Dial(client.Options{
		HostPort: host,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	w := worker.New(c, taskQueue, worker.Options{})

	llmTimeout := config.LLMTimeout
	if llmTimeout == 0 {
		llmTimeout = 2 * time.Minute
	}

	toolTimeout := config.ToolTimeout
	if toolTimeout == 0 {
		toolTimeout = 5 * time.Minute
	}

	executor := &TemporalExecutor{
		client:        c,
		worker:        w,
		taskQueue:     taskQueue,
		modelRegistry: NewModelRegistry(),
		llmTimeout:    llmTimeout,
		toolTimeout:   toolTimeout,
	}

	// Register activities
	w.RegisterActivity(LLMActivity)

	return executor, nil
}

// RegisterAgentTree recursively registers an agent and all its sub-agents and tools
func (e *TemporalExecutor) RegisterAgentTree(agent Agent) {
	e.registerAgent(agent)

	// Register tools
	if th, ok := agent.(ToolHolder); ok {
		for _, t := range th.Tools() {
			if at, ok := t.(*AgentTool); ok {
				// AgentTool -> register as workflow
				e.RegisterAgentTree(at.GetAgent())
			} else {
				// Normal tool -> register as activity
				e.registerTool(t)
			}
		}
	}

	// Recursively register sub-agents
	if sh, ok := agent.(SubAgentHolder); ok {
		for _, sub := range sh.SubAgents() {
			e.RegisterAgentTree(sub)
		}
	}
}

func (e *TemporalExecutor) registerAgent(agent Agent) {
	name := agent.Name(context.Background())
	var workflowFn interface{}

	switch a := agent.(type) {
	case *ChatModelAgent:
		workflowFn = e.buildChatModelWorkflow(a.GetConfig())
	case *SequentialAgent:
		workflowFn = e.buildSequentialWorkflow(a.SubAgents())
	case *ParallelAgent:
		workflowFn = e.buildParallelWorkflow(a.SubAgents())
	case *LoopAgent:
		workflowFn = e.buildLoopWorkflow(a.SubAgents(), a.GetMaxIteration())
	default:
		return
	}

	if workflowFn != nil {
		e.worker.RegisterWorkflowWithOptions(workflowFn, workflow.RegisterOptions{Name: name})
	}
}

func (e *TemporalExecutor) registerTool(t tool.InvokableTool) {
	info, err := t.Info(context.Background())
	if err != nil {
		return
	}

	activityFn := func(ctx context.Context, args string) (string, error) {
		return t.InvokableRun(ctx, args)
	}

	e.worker.RegisterActivityWithOptions(activityFn, activity.RegisterOptions{Name: info.Name})
}

// buildChatModelWorkflow builds a workflow function for ChatModelAgent
func (e *TemporalExecutor) buildChatModelWorkflow(config *ChatModelAgentConfig) interface{} {
	// Identify agent tools
	agentTools := make(map[string]Agent)
	for _, t := range config.ToolsConfig.Tools {
		info, _ := t.Info(context.Background())
		if at, ok := t.(*AgentTool); ok {
			agentTools[info.Name] = at.GetAgent()
		}
	}

	return func(ctx workflow.Context, params WorkflowParams) (*WorkflowResult, error) {
		// Determine root workflow ID for event routing
		rootWorkflowID := params.RootWorkflowID
		if rootWorkflowID == "" {
			rootWorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
		}

		// Helper to send events via SideEffect (won't re-execute on replay)
		sendEvent := func(event *AgentEvent) {
			workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
				GetEventBus().Send(rootWorkflowID, event)
				return nil
			})
		}

		// Resume signal channel
		resumeCh := workflow.GetSignalChannel(ctx, "resume")

		messages := params.Messages

		// Helper function to wait for resume
		waitForResume := func(interruptInfo any) *schema.Message {
			checkpointID := workflow.GetInfo(ctx).WorkflowExecution.ID
			sendEvent(&AgentEvent{
				Type: "interrupted",
				Action: &AgentAction{Interrupted: true},
				InterruptInfo: &InterruptInfo{
					CheckpointID: checkpointID,
					Info:         interruptInfo,
				},
			})

			var resumeInput ResumeInput
			resumeCh.Receive(ctx, &resumeInput)
			return resumeInput.Message
		}

		// Check for pending resume at start
		_ = waitForResume

		for i := 0; i < config.MaxIter; i++ {
			// Emit llm_start event
			sendEvent(&AgentEvent{
				AgentName: config.Name,
				Type:      "llm_start",
			})

			// Call LLM activity
			activityOpts := workflow.ActivityOptions{
				StartToCloseTimeout: 2 * time.Minute,
			}
			ctx = workflow.WithActivityOptions(ctx, activityOpts)

			var resp LLMResponse
			err := workflow.ExecuteActivity(ctx, "llm_call", LLMRequest{
				Messages:    messages,
				Instruction: config.Instruction,
			}).Get(ctx, &resp)

			if err != nil {
				sendEvent(&AgentEvent{Type: "error", Err: err})
				return nil, err
			}

				// Emit llm_response event
			sendEvent(&AgentEvent{
				AgentName: config.Name,
				Type:      "llm_response",
				Output:    &AgentOutput{Message: schema.AssistantMessage(resp.Content, nil)},
			})

			// No tool calls - return result
			if len(resp.ToolCalls) == 0 {
				sendEvent(&AgentEvent{Type: "completed"})
				return &WorkflowResult{
					Output: resp.Content,
				}, nil
			}

			// Process tool calls
			for _, tc := range resp.ToolCalls {
				// Emit tool_start event
				sendEvent(&AgentEvent{
					AgentName: config.Name,
					Type:      "tool_start",
					ToolCall:  &tc,
				})

				var result string

				// Handle built-in tools
				if IsHumanInputTool(tc.Name) {
					// Human input tool -> Interrupt and wait for resume
					var req struct {
						Question string `json:"question"`
					}
					json.Unmarshal([]byte(tc.Arguments), &req)

					// Wait for human input
					resumeMsg := waitForResume(req.Question)
					if resumeMsg != nil {
						result = resumeMsg.Content
					}
				} else if IsExitTool(tc.Name) {
					// Exit tool -> Return immediately
					var req struct {
						Result string `json:"result"`
					}
					json.Unmarshal([]byte(tc.Arguments), &req)

					sendEvent(&AgentEvent{Type: "completed"})
					return &WorkflowResult{
						Output: req.Result,
						Action: &AgentAction{Exit: true},
					}, nil
				} else if subAgent, ok := agentTools[tc.Name]; ok {
					// AgentTool -> Child Workflow
					childOpts := workflow.ChildWorkflowOptions{
						WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID + "/" + tc.Name,
					}
					childCtx := workflow.WithChildOptions(ctx, childOpts)

					var wfResult WorkflowResult
					err := workflow.ExecuteChildWorkflow(childCtx, subAgent.Name(context.Background()),
						WorkflowParams{
							Messages:       parseToolArgsToMessages(tc.Arguments),
							RootWorkflowID: rootWorkflowID,
						},
					).Get(ctx, &wfResult)

					if err != nil {
						result = fmt.Sprintf("error: %v", err)
					} else {
						result = wfResult.Output
					}
				} else if tc.Name == "transferToAgent" {
					// Transfer to another agent
					var req struct {
						AgentName string `json:"agent_name"`
					}
					json.Unmarshal([]byte(tc.Arguments), &req)

					childOpts := workflow.ChildWorkflowOptions{
						WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID + "/transfer/" + req.AgentName,
					}
					childCtx := workflow.WithChildOptions(ctx, childOpts)

					var wfResult WorkflowResult
					err := workflow.ExecuteChildWorkflow(childCtx, req.AgentName,
						WorkflowParams{
							Messages:       messages,
							RootWorkflowID: rootWorkflowID,
						},
					).Get(ctx, &wfResult)

					if err != nil {
						return nil, err
					}

					sendEvent(&AgentEvent{Type: "completed"})
					return &WorkflowResult{
						Output: wfResult.Output,
						Action: wfResult.Action,
					}, nil
				} else {
					// Normal tool -> Activity
					err := workflow.ExecuteActivity(ctx, tc.Name, tc.Arguments).Get(ctx, &result)
					if err != nil {
						result = fmt.Sprintf("error: %v", err)
					}
				}

				// Emit tool_result event
				sendEvent(&AgentEvent{
					AgentName:  config.Name,
					Type:       "tool_result",
					ToolCall:   &tc,
					ToolResult: result,
				})

				// Append tool result to messages
				messages = append(messages, schema.ToolMessage(result, tc.ID))
			}
		}

		sendEvent(&AgentEvent{Type: "completed"})
		return &WorkflowResult{}, nil
	}
}

// buildSequentialWorkflow builds a workflow function for SequentialAgent
func (e *TemporalExecutor) buildSequentialWorkflow(subAgents []Agent) interface{} {
	return func(ctx workflow.Context, params WorkflowParams) (*WorkflowResult, error) {
		// Determine root workflow ID for event routing
		rootWorkflowID := params.RootWorkflowID
		if rootWorkflowID == "" {
			rootWorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
		}

		// Helper to send events via SideEffect (won't re-execute on replay)
		sendEvent := func(event *AgentEvent) {
			workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
				GetEventBus().Send(rootWorkflowID, event)
				return nil
			})
		}

		messages := params.Messages

		for _, sub := range subAgents {
			subName := sub.Name(context.Background())

			childOpts := workflow.ChildWorkflowOptions{
				WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID + "/" + subName,
			}
			childCtx := workflow.WithChildOptions(ctx, childOpts)

			var result WorkflowResult
			err := workflow.ExecuteChildWorkflow(childCtx, subName,
				WorkflowParams{
					Messages:       messages,
					RootWorkflowID: rootWorkflowID,
				},
			).Get(ctx, &result)

			if err != nil {
				return nil, err
			}

			// Check for interrupt
			if result.Action != nil && result.Action.Interrupted {
				return &result, nil
			}

			// Append output to messages
			if result.Output != "" {
				messages = append(messages, schema.AssistantMessage(result.Output, nil))
			}

			// Check for exit
			if result.Action != nil && result.Action.Exit {
				return &result, nil
			}
		}

		sendEvent(&AgentEvent{Type: "completed"})
		return &WorkflowResult{
			Output: getLastOutput(messages),
		}, nil
	}
}

// buildParallelWorkflow builds a workflow function for ParallelAgent
func (e *TemporalExecutor) buildParallelWorkflow(subAgents []Agent) interface{} {
	return func(ctx workflow.Context, params WorkflowParams) (*WorkflowResult, error) {
		// Determine root workflow ID for event routing
		rootWorkflowID := params.RootWorkflowID
		if rootWorkflowID == "" {
			rootWorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
		}

		// Helper to send events via SideEffect (won't re-execute on replay)
		sendEvent := func(event *AgentEvent) {
			workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
				GetEventBus().Send(rootWorkflowID, event)
				return nil
			})
		}

		// Start all child workflows in parallel
		var futures []workflow.ChildWorkflowFuture
		for _, sub := range subAgents {
			subName := sub.Name(context.Background())

			childOpts := workflow.ChildWorkflowOptions{
				WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID + "/" + subName,
			}
			childCtx := workflow.WithChildOptions(ctx, childOpts)

			f := workflow.ExecuteChildWorkflow(childCtx, subName, WorkflowParams{
				Messages:       params.Messages,
				RootWorkflowID: rootWorkflowID,
			})
			futures = append(futures, f)
		}

		// Wait for all to complete
		var results []*WorkflowResult
		for _, f := range futures {
			var result WorkflowResult
			if err := f.Get(ctx, &result); err != nil {
				return nil, err
			}
			results = append(results, &result)
		}

		sendEvent(&AgentEvent{Type: "completed"})
		return &WorkflowResult{
			Output: mergeOutputs(results),
		}, nil
	}
}

// buildLoopWorkflow builds a workflow function for LoopAgent
func (e *TemporalExecutor) buildLoopWorkflow(subAgents []Agent, maxIteration int) interface{} {
	return func(ctx workflow.Context, params WorkflowParams) (*WorkflowResult, error) {
		// Determine root workflow ID for event routing
		rootWorkflowID := params.RootWorkflowID
		if rootWorkflowID == "" {
			rootWorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
		}

		// Helper to send events via SideEffect (won't re-execute on replay)
		sendEvent := func(event *AgentEvent) {
			workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
				GetEventBus().Send(rootWorkflowID, event)
				return nil
			})
		}

		messages := params.Messages

		for i := 0; i < maxIteration; i++ {
			for _, sub := range subAgents {
				subName := sub.Name(context.Background())

				childOpts := workflow.ChildWorkflowOptions{
					WorkflowID: fmt.Sprintf("%s/%s/%d", workflow.GetInfo(ctx).WorkflowExecution.ID, subName, i),
				}
				childCtx := workflow.WithChildOptions(ctx, childOpts)

				var result WorkflowResult
				err := workflow.ExecuteChildWorkflow(childCtx, subName,
					WorkflowParams{
						Messages:       messages,
						RootWorkflowID: rootWorkflowID,
					},
				).Get(ctx, &result)

				if err != nil {
					return nil, err
				}

				if result.Output != "" {
					messages = append(messages, schema.AssistantMessage(result.Output, nil))
				}

				// Check for break loop
				if result.Action != nil && result.Action.BreakLoop {
					sendEvent(&AgentEvent{Type: "completed"})
					return &result, nil
				}
			}
		}

		sendEvent(&AgentEvent{Type: "completed"})
		return &WorkflowResult{
			Output: getLastOutput(messages),
		}, nil
	}
}

// StartWorkflow starts a workflow for the given agent
func (e *TemporalExecutor) StartWorkflow(ctx context.Context, agentName string, input *AgentInput) (string, error) {
	workflowID := fmt.Sprintf("agent-%s-%d", agentName, time.Now().UnixNano())

	_, err := e.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: e.taskQueue,
	}, agentName, WorkflowParams{Messages: input.Messages})

	if err != nil {
		return "", err
	}

	return workflowID, nil
}

// SubscribeEvents subscribes to events from a running workflow via EventBus
func (e *TemporalExecutor) SubscribeEvents(ctx context.Context, workflowID string) <-chan *AgentEvent {
	// Register with EventBus
	eventBus := GetEventBus()
	eventCh := eventBus.Register(workflowID)

	// Create output channel
	outputCh := make(chan *AgentEvent, 100)

	go func() {
		defer close(outputCh)
		defer eventBus.Unregister(workflowID)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				outputCh <- event
				if event.Type == "completed" || event.Type == "error" {
					return
				}
			}
		}
	}()

	return outputCh
}

// Resume sends a resume signal to an interrupted workflow
func (e *TemporalExecutor) Resume(ctx context.Context, workflowID string, input *ResumeInput) error {
	return e.client.SignalWorkflow(ctx, workflowID, "", "resume", input)
}

// Start starts the worker
func (e *TemporalExecutor) Start() error {
	return e.worker.Start()
}

// Stop stops the worker and closes the client
func (e *TemporalExecutor) Stop() {
	e.worker.Stop()
	e.client.Close()
}

// Helper functions

func parseToolArgsToMessages(args string) []*schema.Message {
	var req struct {
		Message string `json:"message"`
	}
	json.Unmarshal([]byte(args), &req)
	return []*schema.Message{schema.UserMessage(req.Message)}
}

func getLastOutput(messages []*schema.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == schema.Assistant {
			return messages[i].Content
		}
	}
	return ""
}

func mergeOutputs(results []*WorkflowResult) string {
	var outputs []string
	for _, r := range results {
		if r.Output != "" {
			outputs = append(outputs, r.Output)
		}
	}
	if len(outputs) == 1 {
		return outputs[0]
	}
	result, _ := json.Marshal(outputs)
	return string(result)
}
