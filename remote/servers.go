// Copyright (C) 2022-2023 Rafael Galvan <rafael.galvan@rigsofrods.org>

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

package remote

import (
	"carbon/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"sync"

	"github.com/apex/log"
	"golang.org/x/sync/errgroup"
)

func (c *client) GetServers(ctx context.Context) ([]domain.Server, error) {
	servers, meta, err := c.getServersPaged(ctx, 1)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	if meta.LastPage > 1 {
		g, ctx := errgroup.WithContext(ctx)
		for page := meta.CurrentPage + 1; page <= meta.LastPage; page++ {
			page := page
			g.Go(func() error {
				p, _, err := c.getServersPaged(ctx, int(page))
				if err != nil {
					return err
				}
				mu.Lock()
				servers = append(servers, p...)
				mu.Unlock()
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
	}
	return servers, nil
}

func (c *client) CreateServer(ctx context.Context, server domain.Server) (domain.Server, error) {
	_, ok := ctx.Value("Authorization").(string)
	if !ok {
		panic("remote/servers: cannot extract authorization token: not present in request context")
	}

	var dataMap map[string]interface{}

	data, err := json.Marshal(server)
	if err != nil {
		panic("remote/servers: could not marshal data")
	}

	err = json.Unmarshal(data, &dataMap)
	if err != nil {
		panic("remote/servers: could not unmarshal data")
	}

	keyValues := url.Values{}
	for key, value := range dataMap {
		keyValues.Add(key, fmt.Sprintf("%v", value))
	}

	_, err = c.Post(ctx, "/servers", keyValues, nil)
	if err != nil {
		return domain.Server{}, err
	}

	return server, nil
}

func (c *client) getServersPaged(ctx context.Context, page int) ([]domain.Server, Pagination, error) {
	var r struct {
		Data []domain.Server `json:"servers"`
		Meta Pagination      `json:"pagination"`
	}

	res, err := c.Get(ctx, "/servers", q{"page": strconv.Itoa(page)}, nil)
	if err != nil {
		return nil, r.Meta, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithField("error", err).Error("")
		}
	}(res.Body)
	if err := res.BindJSON(&r); err != nil {
		return nil, r.Meta, err
	}
	return r.Data, r.Meta, nil
}
