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

package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// InvokeFunc is the function type for the tool
type InvokeFunc[T, D any] func(ctx context.Context, input T) (output D, err error)

// InferTool creates an InvokableTool from a given function by inferring the ToolInfo from the function's request parameters
func InferTool[T, D any](toolName, toolDesc string, fn InvokeFunc[T, D]) (InvokableTool, error) {
	// Generate JSON schema from T
	paramsOneOf, err := goStruct2ParamsOneOf[T]()
	if err != nil {
		return nil, err
	}

	info := &schema.ToolInfo{
		Name:        toolName,
		Desc:        toolDesc,
		ParamsOneOf: paramsOneOf,
	}

	return &invokableTool[T, D]{
		info: info,
		fn:   fn,
	}, nil
}

type invokableTool[T, D any] struct {
	info *schema.ToolInfo
	fn   InvokeFunc[T, D]
}

func (t *invokableTool[T, D]) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *invokableTool[T, D]) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error) {
	// Unmarshal input
	var input T
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	// Invoke function
	output, err := t.fn(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to invoke tool %s: %w", t.info.Name, err)
	}

	// Marshal output
	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(result), nil
}

// goStruct2ParamsOneOf converts a go struct to ParamsOneOf
func goStruct2ParamsOneOf[T any]() (*schema.ParamsOneOf, error) {
	// Create params from struct fields
	// In production, use reflection to properly extract field info
	params := make(map[string]*schema.ParameterInfo)

	// For now, return empty params - in production should use reflection
	// to extract field names and types from the struct
	return schema.NewParamsOneOfByParams(params), nil
}
