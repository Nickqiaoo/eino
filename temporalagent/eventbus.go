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
	"sync"
)

// EventBus manages event channels for workflows
type EventBus struct {
	mu       sync.RWMutex
	channels map[string]chan *AgentEvent
}

// Global event bus
var globalEventBus = &EventBus{
	channels: make(map[string]chan *AgentEvent),
}

// GetEventBus returns the global event bus
func GetEventBus() *EventBus {
	return globalEventBus
}

// Register creates a channel for a workflow
func (b *EventBus) Register(workflowID string) <-chan *AgentEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *AgentEvent, 100)
	b.channels[workflowID] = ch
	return ch
}

// Unregister removes and closes a workflow's channel
func (b *EventBus) Unregister(workflowID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.channels[workflowID]; ok {
		close(ch)
		delete(b.channels, workflowID)
	}
}

// Send sends an event to a workflow's channel
func (b *EventBus) Send(workflowID string, event *AgentEvent) {
	b.mu.RLock()
	ch, ok := b.channels[workflowID]
	b.mu.RUnlock()

	if ok {
		select {
		case ch <- event:
		default:
			// Channel full, drop event (or could log)
		}
	}
}

// SendEventActivity is an activity that sends events to the event bus
func SendEventActivity(workflowID string, event *AgentEvent) error {
	globalEventBus.Send(workflowID, event)
	return nil
}
