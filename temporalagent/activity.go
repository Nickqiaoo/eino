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

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// LLMActivityInput is the input for LLM activity
type LLMActivityInput struct {
	Messages    []*schema.Message
	Instruction string
	Tools       []*schema.ToolInfo
	Model       model.ChatModel
}

// LLMActivityOutput is the output from LLM activity
type LLMActivityOutput struct {
	Content   string
	ToolCalls []ToolCall
}

// LLMActivity executes LLM call
func LLMActivity(ctx context.Context, input LLMActivityInput) (*LLMActivityOutput, error) {
	if input.Model == nil {
		return &LLMActivityOutput{
			Content: "No model configured",
		}, nil
	}

	// Prepare messages with instruction
	messages := input.Messages
	if input.Instruction != "" && len(messages) > 0 {
		// Add system instruction as first message if not present
		hasSystem := false
		for _, msg := range messages {
			if msg.Role == schema.System {
				hasSystem = true
				break
			}
		}
		if !hasSystem {
			systemMsg := &schema.Message{
				Role:    schema.System,
				Content: input.Instruction,
			}
			messages = append([]*schema.Message{systemMsg}, messages...)
		}
	}

	// Prepare options
	var opts []model.Option
	if len(input.Tools) > 0 {
		opts = append(opts, model.WithTools(input.Tools))
	}

	// Call model
	resp, err := input.Model.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	// Convert response
	output := &LLMActivityOutput{
		Content: resp.Content,
	}

	// Convert tool calls
	for _, tc := range resp.ToolCalls {
		output.ToolCalls = append(output.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return output, nil
}

// ModelRegistry holds model instances for activities
type ModelRegistry struct {
	models map[string]model.ChatModel
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make(map[string]model.ChatModel),
	}
}

// Register registers a model with a name
func (r *ModelRegistry) Register(name string, m model.ChatModel) {
	r.models[name] = m
}

// Get retrieves a model by name
func (r *ModelRegistry) Get(name string) model.ChatModel {
	return r.models[name]
}
