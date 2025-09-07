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
	"sync"

	"github.com/apex/log"
)

type Manager struct {
	mu        sync.RWMutex
	resources []*domain.Resource
	client    remote.Client
}

// NewManager creates a new instance of Manager, initializes it using the provided
// context, and returns it. The function takes a context and a remote.Client as
// parameters. If initialization fails, it returns an error.
//
// Parameters:
//   - ctx: The context to control cancellation and deadlines.
//   - client: The remote.Client used for communication with external resources.
//
// Returns:
//   - *Manager: A pointer to the newly created Manager instance.
//   - error: An error if initialization fails, otherwise nil.
func NewManager(ctx context.Context, client remote.Client) (*Manager, error) {
	m := &Manager{client: client}
	err := m.init(ctx)
	return m, err
}

func (m *Manager) init(ctx context.Context) error {
	log.Info("fetching resources from remote API...")
	resources, err := m.client.GetResources(ctx)
	if err != nil {
		return err
	}

	for _, data := range resources {
		data := data
		m.Add(&data)
	}

	return nil
}

func (m *Manager) AsyncRefreshCache(ctx context.Context) error {
	log.Info("refreshing resources cache from remote API...")
	resources, err := m.client.GetResources(ctx)
	// This will prevent the cache from being overwritten in case of
	// any HTTP errors.
	if err != nil {
		return err
	}

	var newCache []*domain.Resource
	for _, data := range resources {
		data := data
		newCache = append(newCache, &data)
	}

	m.Put(newCache)

	return nil
}

// Put replaces the current list of resources managed by the Manager with the provided slice of resources.
// It ensures thread-safety by acquiring a lock before updating the resources and releasing it afterward.
//
// Parameters:
//
//	r - A slice of pointers to domain.Resource that will replace the current resources.
func (m *Manager) Put(r []*domain.Resource) {
	m.mu.Lock()
	m.resources = r
	m.mu.Unlock()
}

func (m *Manager) Add(r *domain.Resource) {
	m.mu.Lock()
	m.resources = append(m.resources, r)
	m.mu.Unlock()
}

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

func (m *Manager) Collection() []*domain.Resource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resources
}
