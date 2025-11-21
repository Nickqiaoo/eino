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

	"github.com/cloudwego/eino/schema"
)

// Option is the option for tool execution
type Option func(*Options)

// Options contains tool execution options
type Options struct {
	// Add options as needed
}

// BaseTool is the base interface for tools
type BaseTool interface {
	Info(ctx context.Context) (*schema.ToolInfo, error)
}

// InvokableTool is a tool that can be invoked
type InvokableTool interface {
	BaseTool
	InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}
