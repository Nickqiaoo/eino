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
	"sync"

	"github.com/cloudwego/eino/schema"
)

// CallbackContext provides context information for callbacks
type CallbackContext struct {
	AgentName    string
	WorkflowID   string
	InvocationID string
}

// BeforeAgentCallback is called before agent execution
type BeforeAgentCallback func(
	ctx context.Context,
	cbCtx *CallbackContext,
	input []*schema.Message,
) ([]*schema.Message, error)

// AfterAgentCallback is called after agent execution
type AfterAgentCallback func(
	ctx context.Context,
	cbCtx *CallbackContext,
	output *WorkflowResult,
	err error,
) (*WorkflowResult, error)

// BeforeModelCallback is called before LLM call
type BeforeModelCallback func(
	ctx context.Context,
	cbCtx *CallbackContext,
	req *LLMRequest,
) (*LLMResponse, error)

// AfterModelCallback is called after LLM call
type AfterModelCallback func(
	ctx context.Context,
	cbCtx *CallbackContext,
	resp *LLMResponse,
	err error,
) (*LLMResponse, error)

// BeforeToolCallback is called before tool execution
type BeforeToolCallback func(
	ctx context.Context,
	cbCtx *CallbackContext,
	toolInfo *schema.ToolInfo,
	args map[string]any,
) (map[string]any, error)

// AfterToolCallback is called after tool execution
type AfterToolCallback func(
	ctx context.Context,
	cbCtx *CallbackContext,
	toolInfo *schema.ToolInfo,
	args map[string]any,
	result map[string]any,
	err error,
) (map[string]any, error)

// CallbackHandler contains all callback functions
type CallbackHandler struct {
	BeforeAgent BeforeAgentCallback
	AfterAgent  AfterAgentCallback
	BeforeModel BeforeModelCallback
	AfterModel  AfterModelCallback
	BeforeTool  BeforeToolCallback
	AfterTool   AfterToolCallback
}

// CallbackRegistry manages callbacks for different agents
type CallbackRegistry struct {
	mu        sync.RWMutex
	callbacks map[string]*CallbackHandler // agentName -> callbacks
}

// Global callback registry
var globalCallbackRegistry = &CallbackRegistry{
	callbacks: make(map[string]*CallbackHandler),
}

// GetCallbackRegistry returns the global callback registry
func GetCallbackRegistry() *CallbackRegistry {
	return globalCallbackRegistry
}

// Register registers callbacks for an agent
func (r *CallbackRegistry) Register(agentName string, handler *CallbackHandler) {
	if handler == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks[agentName] = handler
}

// Get retrieves callbacks for an agent
func (r *CallbackRegistry) Get(agentName string) *CallbackHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.callbacks[agentName]
}

// Unregister removes callbacks for an agent
func (r *CallbackRegistry) Unregister(agentName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.callbacks, agentName)
}
