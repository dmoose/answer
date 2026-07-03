/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package queue

import (
	"context"
	"sync"

	"github.com/apache/answer/internal/multisite"
	"github.com/segmentfault/pacman/log"
)

type Service[T any] interface {
	// Send enqueues a message to be processed asynchronously.
	Send(ctx context.Context, msg T)

	// RegisterHandler sets the handler function for processing messages.
	RegisterHandler(handler func(ctx context.Context, msg T) error)

	// Close gracefully shuts down the queue, waiting for pending messages to be processed.
	Close()
}

// envelope carries the message plus the site scope of the enqueuing request,
// so handlers run under the site the work originated on instead of an empty
// context that silently de-scopes site-aware reads and writes.
type envelope[T any] struct {
	siteID string
	msg    T
}

// Queue is a generic message queue service that processes messages asynchronously.
// It is thread-safe and supports graceful shutdown.
type Queue[T any] struct {
	name    string
	queue   chan envelope[T]
	handler func(ctx context.Context, msg T) error
	mu      sync.RWMutex
	closed  bool
	wg      sync.WaitGroup
}

// New creates a new queue with the given name and buffer size.
func New[T any](name string, bufferSize int) *Queue[T] {
	q := &Queue[T]{
		name:  name,
		queue: make(chan envelope[T], bufferSize),
	}
	q.startWorker()
	return q
}

// Send enqueues a message to be processed asynchronously.
// It will block if the queue is full.
func (q *Queue[T]) Send(ctx context.Context, msg T) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		log.Warnf("[%s] queue is closed, dropping message", q.name)
		return
	}

	select {
	case q.queue <- envelope[T]{siteID: multisite.SiteIDFromContext(ctx), msg: msg}:
		log.Debugf("[%s] enqueued message: %+v", q.name, msg)
	case <-ctx.Done():
		log.Warnf("[%s] context cancelled while sending message", q.name)
	}
}

// RegisterHandler sets the handler function for processing messages.
// This is thread-safe and can be called at any time.
func (q *Queue[T]) RegisterHandler(handler func(ctx context.Context, msg T) error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handler = handler
}

// Close gracefully shuts down the queue, waiting for pending messages to be processed.
func (q *Queue[T]) Close() {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.closed = true
	q.mu.Unlock()

	close(q.queue)
	q.wg.Wait()
	log.Infof("[%s] queue closed", q.name)
}

// startWorker starts the background goroutine that processes messages.
func (q *Queue[T]) startWorker() {
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for e := range q.queue {
			q.processMessage(e)
		}
	}()
}

// processMessage handles a single message with proper synchronization.
func (q *Queue[T]) processMessage(e envelope[T]) {
	q.mu.RLock()
	handler := q.handler
	q.mu.RUnlock()

	if handler == nil {
		log.Warnf("[%s] no handler registered, dropping message: %+v", q.name, e.msg)
		return
	}

	// Rebuild the enqueuing request's site scope so site-aware reads and
	// writes in handlers keep their attribution.
	ctx := multisite.WithSiteID(context.Background(), e.siteID)
	if err := handler(ctx, e.msg); err != nil {
		log.Errorf("[%s] handler error: %v", q.name, err)
	}
}
