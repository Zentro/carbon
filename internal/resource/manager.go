// Copyright (C) 2022-2023 Rafael Galvan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package resource

import (
	"carbon/domain"
	"carbon/remote"
	"context"
	"log/slog"
	"sync"
	"time"
)

type Manager struct {
	mu        sync.RWMutex
	resources []*domain.Resource
	client    remote.Client
}

// NewManager creates a new instance of Manager and initializes it by fetching resources from the remote API.
func NewManager(ctx context.Context, client remote.Client) (*Manager, error) {
	m := &Manager{client: client}

	if err := m.init(ctx); err != nil {
		return nil, err
	}

	// Start the resource manager in a separate goroutine
	// to handle periodically refreshing the resource cache.
	m.Start(ctx)

	return m, nil
}

func (m *Manager) Start(ctx context.Context) {
	slog.Info("starting resource manager...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("stopping resource manager...")
				return
			case <-time.After(5 * time.Minute):
				if err := m.purgeAndRefresh(ctx); err != nil {
					// Not fatal, just log the error and try again later.
					slog.Error("could not refresh resources cache", "error", err)
				} else {
					slog.Debug("resources cache successfully refreshed")
				}
			}
		}
	}()
}

func (m *Manager) init(ctx context.Context) error {
	slog.Info("fetching resources from remote API...")
	resources, err := m.client.GetResources(ctx)
	if err != nil {
		return err
	}

	for _, data := range resources {
		m.Add(&data)
	}

	return nil
}

// purgeAndRefresh fetches the latest resources from the remote API and updates the internal cache.
func (m *Manager) purgeAndRefresh(ctx context.Context) error {
	slog.Debug("refreshing resources cache...")
	resources, err := m.client.GetResources(ctx)
	// This will prevent the cache from being overwritten in case of
	// any HTTP errors.
	if err != nil {
		return err
	}

	var r []*domain.Resource
	for _, data := range resources {
		data := data
		r = append(r, &data)
	}

	m.Put(r)

	return nil
}

// Put replaces the entire resources cache with the provided slice of resources.
func (m *Manager) Put(r []*domain.Resource) {
	m.mu.Lock()
	m.resources = r
	m.mu.Unlock()
}

// Add appends a new resource to the resources cache.
func (m *Manager) Add(r *domain.Resource) {
	m.mu.Lock()
	m.resources = append(m.resources, r)
	m.mu.Unlock()
}

// Find searches for a resource that matches the provided filter function.
func (m *Manager) Find(filter func(match *domain.Resource) bool) *domain.Resource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.resources {
		if filter(v) {
			return v
		}
	}
	return nil
}

// Collection returns a copy of the entire resources cache.
func (m *Manager) Collection() []*domain.Resource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resources
}
