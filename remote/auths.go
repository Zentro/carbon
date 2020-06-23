package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *client) ValidateUserAuthCredentials(ctx context.Context, data interface{}) (RawUserAuthResponse, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		panic("remote/auths: could not marshal data")
	}

	var dataMap map[string]interface{}

	err = json.Unmarshal(payload, &dataMap)
	if err != nil {
		panic("remote/auths: could not unmarshal data")
	}

	formValues := url.Values{}
	for key, value := range dataMap {
		formValues.Add(key, fmt.Sprintf("%v", value))
	}

	resp, httpErr := c.Post(ctx, "/bridge/auth", formValues, nil)
	if httpErr != nil {
		return RawUserAuthResponse{}, httpErr
	}

	var rawData RawUserAuthResponse

	err = resp.BindJSON(&rawData)
	if err != nil {
		panic("remote/auths: could not bind json data")
	}

	return rawData, nil
}

// extractAuthorization will extract the proper heads passed down from upstream in the context.
func extractAuthorization(ctx context.Context) string {
	token, ok := ctx.Value("Authorization").(string)
	if !ok {
		panic("remote/auths: cannot extract authorization token: not present in request context")
	}
	return token
}
