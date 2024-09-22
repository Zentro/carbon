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
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/apex/log"

	"golang.org/x/sync/errgroup"
)

func (c *client) GetResources(ctx context.Context) ([]domain.Resource, error) {
	resources, meta, err := c.getResourcesPaged(ctx, 1)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	if meta.LastPage > 1 {
		g, ctx := errgroup.WithContext(ctx)
		for page := meta.CurrentPage + 1; page <= meta.LastPage; page++ {
			page := page
			g.Go(func() error {
				p, _, err := c.getResourcesPaged(ctx, int(page))
				if err != nil {
					return err
				}
				mu.Lock()
				resources = append(resources, p...)
				mu.Unlock()
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
	}
	return resources, nil
}

func (c *client) GetResource(ctx context.Context, rid string) (domain.Resource, error) {
	var r struct {
		Data domain.Resource `json:"resource"`
	}

	res, err := c.Get(ctx, fmt.Sprintf("/resources/%s", rid), nil, nil)
	if err != nil {
		return domain.Resource{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithField("error", err).Error("")
		}
	}(res.Body)
	if err := res.BindJSON(&r); err != nil {
		return domain.Resource{}, err
	}
	return r.Data, nil
}

func (c *client) GetResourceCategories(ctx context.Context) ([]domain.ResourceCategory, TreeMap, error) {
	var r struct {
		Data []domain.ResourceCategory `json:"categories"`
		//Meta TreeMap                   `json:"tree_map"`
	}

	res, err := c.Get(ctx, "/resource-categories", nil, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithField("error", err).Error("")
		}
	}(res.Body)
	if err := res.BindJSON(&r); err != nil {
		return nil, nil, err
	}
	return r.Data, nil, nil
}

func (c *client) GetResourceReviews(ctx context.Context, rid string) ([]domain.ResourceReview, error) {
	reviews, meta, err := c.getResourceReviewsPaged(ctx, 1, rid)
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	if meta.LastPage > 1 {
		g, ctx := errgroup.WithContext(ctx)
		for page := meta.CurrentPage + 1; page <= meta.LastPage; page++ {
			page := page
			g.Go(func() error {
				p, _, err := c.getResourceReviewsPaged(ctx, int(page), rid)
				if err != nil {
					return err
				}
				mu.Lock()
				reviews = append(reviews, p...)
				mu.Unlock()
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
	}
	return reviews, nil
}

func (c *client) GetResourceVersions(ctx context.Context, rid string) ([]domain.ResourceVersion, error) {
	var r struct {
		Data []domain.ResourceVersion `json:"versions"`
	}
	res, err := c.Get(ctx, fmt.Sprintf("/resources/%s/versions", rid), nil, nil)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithField("error", err).Error("")
		}
	}(res.Body)
	if err := res.BindJSON(&r); err != nil {
		return nil, err
	}
	return r.Data, nil
}

func (c *client) GetResourceVersion(ctx context.Context, vid string) (domain.ResourceVersion, error) {
	var r struct {
		Data domain.ResourceVersion `json:"version"`
	}
	res, err := c.Get(ctx, fmt.Sprintf("/resource-versions/%s", vid), nil, nil)
	if err != nil {
		return domain.ResourceVersion{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithField("error", err).Error("")
		}
	}(res.Body)
	if err := res.BindJSON(&r); err != nil {
		return domain.ResourceVersion{}, err
	}
	return r.Data, nil
}

func (c *client) GetResourceCategory(ctx context.Context) (domain.ResourceCategory, error) {
	return domain.ResourceCategory{}, nil
}

func (c *client) getResourceReviewsPaged(ctx context.Context, page int, rid string) ([]domain.ResourceReview, Pagination, error) {
	var r struct {
		Data []domain.ResourceReview `json:"reviews"`
		Meta Pagination              `json:"pagination"`
	}

	res, err := c.Get(ctx, fmt.Sprintf("/resources/%s/reviews", rid), q{"page": strconv.Itoa(page)}, nil)
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

func (c *client) getResourcesPaged(ctx context.Context, page int) ([]domain.Resource, Pagination, error) {
	var r struct {
		Data []domain.Resource `json:"resources"`
		Meta Pagination        `json:"pagination"`
	}

	res, err := c.Get(ctx, "/resources", q{"page": strconv.Itoa(page)}, nil)
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
