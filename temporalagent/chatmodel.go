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
	"github.com/cloudwego/eino/temporalagent/tool"
)

// ChatModelAgentConfig is the configuration for ChatModelAgent
type ChatModelAgentConfig struct {
	Name        string
	Description string
	Instruction string
	Model       model.ChatModel
	ToolsConfig ToolsConfig
	SubAgents   []Agent // for transfer
	MaxIter     int
}

// ChatModelAgent implements Agent interface
type ChatModelAgent struct {
	config *ChatModelAgentConfig
}

// NewChatModelAgent creates a new ChatModelAgent
func NewChatModelAgent(ctx context.Context, config *ChatModelAgentConfig) (Agent, error) {
	if config.MaxIter == 0 {
		config.MaxIter = 10
	}

	return &ChatModelAgent{
		config: config,
	}, nil
}

func (a *ChatModelAgent) Name(ctx context.Context) string {
	return a.config.Name
}

func (a *ChatModelAgent) Description(ctx context.Context) string {
	return a.config.Description
}

func (a *ChatModelAgent) SubAgents() []Agent {
	return a.config.SubAgents
}

func (a *ChatModelAgent) Tools() []tool.InvokableTool {
	return a.config.ToolsConfig.Tools
}

func (a *ChatModelAgent) GetConfig() *ChatModelAgentConfig {
	return a.config
}
