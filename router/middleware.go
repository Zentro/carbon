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

package router

import (
	"carbon/domain"
	"carbon/internal/api_key"
	"carbon/internal/resource"
	"carbon/internal/server"
	"carbon/internal/token"
	"carbon/internal/user"
	"carbon/remote"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AttachCorsHeaders attaches access control headers to all requests.
func AttachCorsHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Content-Encoding, Accept-Encoding, Authorization")

		// Around 2 hours, which is allowable by most browsers including Chromium.
		// @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Max-Age#Directives
		c.Header("Access-Control-Max-Age", "7200")
		c.Next()
	}
}

func AttachApiKeyManager(m *api_key.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("api_key_manager", m)
		c.Next()
	}
}

func AttachApiClient(client remote.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("api_client", client)
		c.Next()
	}
}

func AttachResourceManager(m *resource.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("resource_manager", m)
		c.Next()
	}
}

func AttachUserManager(m *user.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_manager", m)
		c.Next()
	}
}

func AttachServerManager(m *server.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("server_manager", m)
		c.Next()
	}
}

func AttachTokenManager(m *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("token_manager", m)
		c.Next()
	}
}

func ExtractApiKeyManager(c *gin.Context) *api_key.Manager {
	if v, ok := c.Get("api_key_manager"); ok {
		return v.(*api_key.Manager)
	}
	panic("router/middleware: api key manager not present in context")
}

// ExtractApiClient returns the remote API client instance and set it into the
// gin.Context
func ExtractApiClient(c *gin.Context) remote.Client {
	if v, ok := c.Get("api_client"); ok {
		return v.(remote.Client)
	}
	panic("router/middleware: remote client not present in context")
}

// ExtractResourceManager returns the resource manager instance and set it into
// the gin.Context.
func ExtractResourceManager(c *gin.Context) *resource.Manager {
	if v, ok := c.Get("resource_manager"); ok {
		return v.(*resource.Manager)
	}
	panic("router/middleware: resource manager not present in context")
}

// ExtractServerManager returns the server manager instance and set it into the
// gin.Context.
func ExtractServerManager(c *gin.Context) *server.Manager {
	if v, ok := c.Get("server_manager"); ok {
		return v.(*server.Manager)
	}
	panic("router/middleware: server manager not present in context")
}

// ExtractTokenManager returns the token manager instance and set it into the
// gin.Context
func ExtractTokenManager(c *gin.Context) *token.Manager {
	if v, ok := c.Get("token_manager"); ok {
		return v.(*token.Manager)
	}
	panic("router/middleware: token manager not presnet in context")
}

// ResourceExists will ensure that the request resource exists in our cache.
// Returns a 404 if we cannot locate it. If the resource is found it is set into
// the request context.
func ResourceExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		var r *domain.Resource
		if c.Param("resource") != "" {
			manager := ExtractResourceManager(c)
			r = manager.Find(func(r *domain.Resource) bool {
				return c.Param("resource") == r.ID()
			})
		}
		if r == nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found."})
			return
		}
		c.Set("resource", r)
		c.Next()
	}
}

// ServerExists will ensure that the request server exists in the database.
// Returns a 404 if we cannot locate it. If the resource is found it is set into
// the request context.
func ServerExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param("server") != "" {
			var s *domain.Server
			manager := ExtractServerManager(c)
			serverId, _ := strconv.Atoi(c.Param("server"))
			s, err := manager.FindByID(serverId)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found."})
				return
			}
			c.Set("server", s)
		}
		c.Next()
	}
}

// ExtractResource will return the resource from the gin.Context or panic if
// it is not present.
func ExtractResource(c *gin.Context) *domain.Resource {
	v, ok := c.Get("resource")
	if !ok {
		panic("router/middleware: cannot extract resource: not present in request context")
	}
	return v.(*domain.Resource)
}

// ExtractServer will return the server from the gin.Context or panic if it
// is not present.
func ExtractServer(c *gin.Context) *domain.Server {
	v, ok := c.Get("server")
	if !ok {
		panic("router/middleware: cannot extract server: not present in request context")
	}
	return v.(*domain.Server)
}

// ExtractUser will return the user from the gin.Context or panic if it is
// not present.
func ExtractUser(c *gin.Context) domain.User {
	v, ok := c.Get("user")
	if !ok {
		panic("router/middleware: cannot extract user: not present in request context")
	}
	return v.(domain.User)
}

// ExtractToken will return the token from the gin.Context or panic if it
// is not present.
func ExtractToken(c *gin.Context) domain.Token {
	v, ok := c.Get("token")
	if !ok {
		panic("router/middleware: cannot extract token: not present in request context")
	}
	return v.(domain.Token)
}

// RequireApiAuthorization will only check if the proper API authentication
// heads are present.
func RequireApiAuthorization() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.SplitN(c.GetHeader("Api-Authorization"), " ", 2)

		if len(key) != 2 || key[0] != "Bearer" {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "The required authorization heads were not present in the request.",
			})

			return
		}

		var r domain.ApiKey
		manager := ExtractApiKeyManager(c)
		r, dbErr := manager.FindByKey(key[1])
		if dbErr != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "You are not authorized to access this endpoint.",
			})

			return
		}

		// Pass the role and an entire copy of the key further up the context.
		c.Set("apiRole", r.Role)
		c.Set("apiKey", r)
	}
}

// RoleRequired will check if the required role matches the role present in the context.
func RoleRequired(required Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleAny, exists := c.Get("apiRole")
		if exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "The required authorization heads were not present in the request.",
			})

			return
		}

		roleStr, ok := roleAny.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "The role type in the authorization heads are not of the expected type.",
			})

			return
		}

		role := Role(roleStr)

		if !role.IsValid() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "The role in the authorization heads could not be determined.",
			})

			return
		}

		if role.IsOperator() {
			c.Next()
			return
		}

		if role != required {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "You are not authorized to access this endpoint.",
			})

			return
		}

		c.Next()
	}
}

// RequireAuthorization will only check if the proper authentication heads
// are present.
func RequireAuthorization() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.SplitN(c.GetHeader("Authorization"), " ", 2)

		if len(token) != 2 || token[0] != "Bearer" {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "The required authorization heads were not present in the request.",
			})

			return
		}

		var r domain.Token
		manager := ExtractTokenManager(c)
		r, dbErr := manager.FindByToken(token[1])
		if dbErr != nil || time.Now().After(r.LoginTokenExpiresAt) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "You are not authorized to access this endpoint.",
			})

			return
		}

		if c.ClientIP() != r.IPAddress {
			NewError(ErrIpMismatch).Abort(c)
			return
		}

		var u domain.User
		u, httpErr := ExtractApiClient(c).GetUser(c, r.UserID)
		if httpErr != nil {
			NewError(httpErr).Abort(c)
			return
		}

		// Pass up further along the context.
		c.Set("user", u)
		c.Set("token", r)

		c.Next()
	}
}

func ExtractAuthorization(c *gin.Context) string {
	v, ok := c.Get("Authorization")
	if !ok {
		panic("router/middleware: cannot extract authorization token: not present in request context")
	}
	token, _ := v.(string)
	return token
}
