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

package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	ta "github.com/cloudwego/eino/temporalagent"
	"github.com/cloudwego/eino/temporalagent/tool"
)

// Define tool input/output types
type SearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type SearchOutput struct {
	Results []string `json:"results"`
}

// Define a search function
func searchFunc(ctx context.Context, input SearchInput) (SearchOutput, error) {
	// Simulate search
	results := []string{
		fmt.Sprintf("Result 1 for: %s", input.Query),
		fmt.Sprintf("Result 2 for: %s", input.Query),
	}
	if input.Limit > 0 && input.Limit < len(results) {
		results = results[:input.Limit]
	}
	return SearchOutput{Results: results}, nil
}

// Calculator tool
type CalcInput struct {
	A  int    `json:"a"`
	B  int    `json:"b"`
	Op string `json:"op"`
}

type CalcOutput struct {
	Result int `json:"result"`
}

func calcFunc(ctx context.Context, input CalcInput) (CalcOutput, error) {
	var result int
	switch input.Op {
	case "add":
		result = input.A + input.B
	case "sub":
		result = input.A - input.B
	case "mul":
		result = input.A * input.B
	case "div":
		if input.B != 0 {
			result = input.A / input.B
		}
	}
	return CalcOutput{Result: result}, nil
}

func main() {
	ctx := context.Background()

	// 1. Create tools using InferTool
	searchTool, err := tool.InferTool("search", "Search for information", searchFunc)
	if err != nil {
		panic(err)
	}

	calcTool, err := tool.InferTool("calculator", "Perform calculations", calcFunc)
	if err != nil {
		panic(err)
	}

	// 2. Create a research agent
	researchAgent, err := ta.NewChatModelAgent(ctx, &ta.ChatModelAgentConfig{
		Name:        "researcher",
		Description: "Expert at researching information",
		Instruction: "You are a research assistant. Use the search tool to find information.",
		Model:       nil, // Replace with actual LLM
		ToolsConfig: ta.ToolsConfig{
			Tools: []tool.InvokableTool{searchTool},
		},
		MaxIter: 5,
	})
	if err != nil {
		panic(err)
	}

	// 3. Wrap research agent as a tool
	researchTool := ta.NewAgentTool(ctx, researchAgent)

	// 4. Create main agent with both tools
	mainAgent, err := ta.NewChatModelAgent(ctx, &ta.ChatModelAgentConfig{
		Name:        "assistant",
		Description: "A helpful assistant",
		Instruction: "You are a helpful assistant. You can search for information and perform calculations.",
		Model:       nil, // Replace with actual LLM
		ToolsConfig: ta.ToolsConfig{
			Tools: []tool.InvokableTool{
				researchTool, // Agent as Tool
				calcTool,     // Normal tool
			},
		},
		MaxIter: 10,
	})
	if err != nil {
		panic(err)
	}

	// 5. Create runner
	runner, err := ta.NewRunner(ctx, mainAgent, &ta.RunnerConfig{
		TaskQueue: "my-agent-tasks",
	})
	if err != nil {
		panic(err)
	}
	defer runner.Close()

	// 6. Run agent
	eventCh := runner.Run(ctx, &ta.AgentInput{
		Messages: []*schema.Message{
			schema.UserMessage("Search for information about AI and calculate 10 + 20"),
		},
	})

	// 7. Process events
	for event := range eventCh {
		switch event.Type {
		case "llm_start":
			fmt.Println("Thinking...")
		case "llm_response":
			if event.Output != nil {
				fmt.Printf("Assistant: %s\n", event.Output.Message.Content)
			}
		case "tool_start":
			fmt.Printf("Calling tool: %s\n", event.ToolCall.Name)
		case "tool_result":
			fmt.Printf("Tool result: %s\n", event.ToolResult)
		case "completed":
			fmt.Println("Done!")
		case "error":
			fmt.Printf("Error: %v\n", event.Err)
		}
	}
}

// Example: Sequential Agent
func exampleSequentialAgent() {
	ctx := context.Background()

	// Create planner agent
	planner, _ := ta.NewChatModelAgent(ctx, &ta.ChatModelAgentConfig{
		Name:        "planner",
		Description: "Creates execution plans",
		Instruction: "You are a planner. Create a step-by-step plan.",
		Model:       nil,
	})

	// Create executor agent
	executor, _ := ta.NewChatModelAgent(ctx, &ta.ChatModelAgentConfig{
		Name:        "executor",
		Description: "Executes plans",
		Instruction: "You are an executor. Execute the given plan step by step.",
		Model:       nil,
	})

	// Create sequential pipeline
	pipeline, _ := ta.NewSequentialAgent(ctx, &ta.SequentialAgentConfig{
		Name:      "plan-execute",
		SubAgents: []ta.Agent{planner, executor},
	})

	// Create runner and execute
	runner, _ := ta.NewRunner(ctx, pipeline, nil)
	defer runner.Close()

	eventCh := runner.Run(ctx, &ta.AgentInput{
		Messages: []*schema.Message{
			schema.UserMessage("Build a web scraper"),
		},
	})

	for event := range eventCh {
		fmt.Printf("Event: %s\n", event.Type)
	}
}

// Example: Parallel Agent
func exampleParallelAgent() {
	ctx := context.Background()

	// Create multiple analysis agents
	techAnalyst, _ := ta.NewChatModelAgent(ctx, &ta.ChatModelAgentConfig{
		Name:        "tech_analyst",
		Description: "Technical analysis",
		Model:       nil,
	})

	marketAnalyst, _ := ta.NewChatModelAgent(ctx, &ta.ChatModelAgentConfig{
		Name:        "market_analyst",
		Description: "Market analysis",
		Model:       nil,
	})

	// Run in parallel
	parallelAgent, _ := ta.NewParallelAgent(ctx, &ta.ParallelAgentConfig{
		Name:      "parallel-analysis",
		SubAgents: []ta.Agent{techAnalyst, marketAnalyst},
	})

	runner, _ := ta.NewRunner(ctx, parallelAgent, nil)
	defer runner.Close()

	eventCh := runner.Run(ctx, &ta.AgentInput{
		Messages: []*schema.Message{
			schema.UserMessage("Analyze the AI industry"),
		},
	})

	for event := range eventCh {
		fmt.Printf("Event: %s from %s\n", event.Type, event.AgentName)
	}
}

// Example: Loop Agent
func exampleLoopAgent() {
	ctx := context.Background()

	// Create an agent that iterates
	worker, _ := ta.NewChatModelAgent(ctx, &ta.ChatModelAgentConfig{
		Name:        "worker",
		Description: "Iterative worker",
		Model:       nil,
	})

	// Loop until condition or max iterations
	loopAgent, _ := ta.NewLoopAgent(ctx, &ta.LoopAgentConfig{
		Name:         "loop-worker",
		SubAgents:    []ta.Agent{worker},
		MaxIteration: 5,
	})

	runner, _ := ta.NewRunner(ctx, loopAgent, nil)
	defer runner.Close()

	eventCh := runner.Run(ctx, &ta.AgentInput{
		Messages: []*schema.Message{
			schema.UserMessage("Refine this code until it's perfect"),
		},
	})

	for event := range eventCh {
		fmt.Printf("Event: %s\n", event.Type)
	}
}
