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
	"reflect"
	"strings"

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

// goStruct2ParamsOneOf converts a go struct to ParamsOneOf using reflection
func goStruct2ParamsOneOf[T any]() (*schema.ParamsOneOf, error) {
	var t T
	typ := reflect.TypeOf(t)

	// Handle pointer types
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %s", typ.Kind())
	}

	params := make(map[string]*schema.ParameterInfo)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag name
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		// Get description from jsonschema tag
		desc := field.Tag.Get("jsonschema")
		if desc != "" {
			// Parse "description=xxx" format
			for _, part := range strings.Split(desc, ",") {
				if strings.HasPrefix(part, "description=") {
					desc = strings.TrimPrefix(part, "description=")
					break
				}
			}
		}

		// Convert Go type to schema type
		paramType := goTypeToSchemaType(field.Type)

		// Check if required (simple heuristic: non-pointer types are required)
		required := field.Type.Kind() != reflect.Ptr

		params[fieldName] = &schema.ParameterInfo{
			Type:     paramType,
			Desc:     desc,
			Required: required,
		}
	}

	return schema.NewParamsOneOfByParams(params), nil
}

// goTypeToSchemaType converts a Go reflect.Type to schema.DataType
func goTypeToSchemaType(t reflect.Type) schema.DataType {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return schema.String
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return schema.Integer
	case reflect.Float32, reflect.Float64:
		return schema.Number
	case reflect.Bool:
		return schema.Boolean
	case reflect.Slice, reflect.Array:
		return schema.Array
	case reflect.Map, reflect.Struct:
		return schema.Object
	default:
		return schema.String
	}
}
