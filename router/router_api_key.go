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
	"net/http"
	"strings"

	"carbon/domain"
	"carbon/internal/api_key"

	"github.com/gin-gonic/gin"
)

// getAllApiKeys returns the caller's API keys, or every key in the system
// for an admin principal. Admin status is the wildcard scope, so this is
// just a single branch on principal.IsAdmin().
//
//	@Tags		api_key
//	@Produce	json
//	@Success	200	{object}	[]domain.ApiKey
//	@Router		/api-keys [get]
func getAllApiKeys(c *gin.Context) {
	p := ExtractPrincipal(c)
	mgr := ExtractApiKeyManager(c)

	var keys []*domain.ApiKey
	var err error
	if p.IsAdmin() {
		keys, err = mgr.Collection()
	} else {
		keys, err = mgr.FindByUser(p.UserID)
	}
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	for _, k := range keys {
		k.Key = ""
	}
	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

// getApiKey returns one API key. The plaintext Key is redacted; it was
// only ever returned at creation time.
//
//	@Tags		api_key
//	@Produce	json
//	@Param		id	path		int	true	"API Key ID"
//	@Success	200	{object}	domain.ApiKey
//	@Router		/api-keys/{id} [get]
func getApiKey(c *gin.Context) {
	k := extractTargetApiKey(c)
	redacted := *k
	redacted.Key = ""
	c.JSON(http.StatusOK, gin.H{"api_key": redacted})
}

// createApiKeyRequest is the body of POST /api-keys. Users create personal,
// unbound keys this way; bound keys are issued by server-creation only.
type createApiKeyRequest struct {
	// Name is a human-readable label shown in the dashboard. Required.
	Name string `json:"name" binding:"required"`
	// Scopes is the explicit subset of the caller's authority this key
	// will carry. Empty means "inherit the full role" — useful for a
	// general-purpose personal token. Wildcards are rejected; pass
	// concrete actions only.
	Scopes []string `json:"scopes"`
}

// postCreateApiKey godoc
//
//	@Tags		api_key
//	@Accept		json
//	@Produce	json
//	@Param		body	body		createApiKeyRequest	true	"Key request"
//	@Success	201		{object}	domain.ApiKey
//	@Router		/api-keys [post]
func postCreateApiKey(c *gin.Context) {
	p := ExtractPrincipal(c)

	var req createApiKeyRequest
	if err := c.BindJSON(&req); err != nil {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Name is required."})
		return
	}

	requested := domain.NewScopeSet(req.Scopes...)
	for scope := range requested {
		if strings.Contains(scope, "*") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Wildcard scopes are not permitted on user-issued keys.",
			})
			return
		}
	}
	if !requested.IsSubsetOf(p.Role.Scopes()) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Requested scopes exceed your role's authority.",
		})
		return
	}

	plaintext, err := api_key.GenerateRandomKey()
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	row := &domain.ApiKey{
		UserID:  p.UserID,
		Name:    req.Name,
		Key:     plaintext,
		Scopes:  requested,
		Enabled: true,
	}
	if err := ExtractApiKeyManager(c).Create(row); err != nil {
		NewError(err).Abort(c)
		return
	}

	// Plaintext returned exactly once. The dashboard should show + copy
	// it to the user, then forget it; the row keeps no plaintext copy
	// readable via subsequent GETs (see getApiKey redaction).
	c.JSON(http.StatusCreated, gin.H{"api_key": row})
}

// deleteApiKey godoc
//
//	@Tags		api_key
//	@Param		id	path	int	true	"API Key ID"
//	@Success	204	"No Content"
//	@Router		/api-keys/{id} [delete]
func deleteApiKey(c *gin.Context) {
	k := extractTargetApiKey(c)
	if err := ExtractApiKeyManager(c).DeleteByID(k.ApiKeyID); err != nil {
		NewError(err).Abort(c)
		return
	}
	c.Status(http.StatusNoContent)
}

// extractTargetApiKey returns the API key loaded into context by
// ApiKeyExists. Panics if the loader middleware was not mounted.
func extractTargetApiKey(c *gin.Context) *domain.ApiKey {
	v, ok := c.Get("apiKey")
	if !ok {
		panic("router/router_api_key: target api key not present; ApiKeyExists middleware missing?")
	}
	return v.(*domain.ApiKey)
}
