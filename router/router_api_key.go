// Copyright (C) 2022-2025 Rafael Galvan

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

package router

import (
	"carbon/domain"
	"carbon/internal/api_key"
	"net/http"

	"github.com/gin-gonic/gin"
)

// getAllApiKeys godoc
//
//	@Tags		api_key
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	[]domain.ApiKey
//	@Failure	400	{object}	RequestError
//	@Failure	404	{object}	RequestError
//	@Failure	500	{object}	RequestError
//	@Router		/api-keys/ [get]
func getAllApiKeys(c *gin.Context) {
	api_keys, err := ExtractApiKeyManager(c).Collection()
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": api_keys,
	})
}

// getApiKey godoc
//
//	@Tags		api_key
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	[]domain.ApiKey
//	@Failure	400	{object}	RequestError
//	@Failure	404	{object}	RequestError
//	@Failure	500	{object}	RequestError
//	@Router		/api-keys/ [get]
func getApiKey(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_key": ExtractApiKeyKey(c),
	})
}

// postCreateApiKey godoc
//
//	@Tags		api_key
//	@Accept		json
//	@Produce	json
//	@Param		apiKey	body		domain.ApiKey	true	"API Key Object"
//	@Success	201		{object}	domain.ApiKey
//	@Failure	400		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/api-keys/ [post]
func postCreateApiKey(c *gin.Context) {
	var newApiKeyRequest domain.ApiKey
	if err := c.BindJSON(&newApiKeyRequest); err != nil {
		return
	}

	randomKey, err := api_key.GenerateRandomKey()
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	// This key should always be random and unique. The SQL unique constraint should
	// prevent non-unique keys from being accepted.
	newApiKeyRequest.Key = randomKey

	manager := ExtractApiKeyManager(c)
	if err := manager.Create(&newApiKeyRequest); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": newApiKeyRequest,
	})
}

// deleteApiKey godoc
//
//	@Tags		api_key
//	@Accept		json
//	@Produce	json
//	@Param		key	path	string	true	"API Key Identifier"
//	@Success	204	"No Content"
//	@Failure	400	{object}	RequestError
//	@Failure	404	{object}	RequestError
//	@Failure	500	{object}	RequestError
//	@Router		/api-keys/{key} [delete]
func deleteApiKey(c *gin.Context) {
	api_key_key := ExtractApiKeyKey(c)
	manager := ExtractApiKeyManager(c)
	if err := manager.Delete(api_key_key.Key); err != nil {
		return
	}

	c.Status(http.StatusNoContent)
}
