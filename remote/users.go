package remote

import (
	"carbon/domain"
	"context"
	"fmt"
	"io"

	"github.com/apex/log"
)

func (c *client) GetUser(ctx context.Context, uid int) (domain.User, error) {
	var r struct {
		Data domain.User `json:"user"`
	}
	res, err := c.Get(ctx, fmt.Sprintf("/users/%d", uid), nil, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.WithField("error", err).Error("")
		}
	}(res.Body)
	if err := res.BindJSON(&r); err != nil {
		return domain.User{}, err
	}
	return r.Data, nil
}
