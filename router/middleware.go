// Copyright (C) 2022, 2025 Rafael Galvan <rafael.galvan@rigsofrods.org>

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
	"carbon/internal/api_login_key"
	"carbon/internal/client"
	"carbon/internal/resource"
	"carbon/internal/server"
	"carbon/internal/user"
	"carbon/remote"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuthContext struct {
	User domain.User `json:"user,omitempty"`
}

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

// AttachApiKeyManager attaches the API key manager instance and set it into
// the gin.Context
func AttachApiKeyManager(m *api_key.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("api_key_manager", m)
		c.Next()
	}
}

func AttachClientManager(m *client.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("client_manager", m)
		c.Next()
	}
}

// AttachApiClient attaches the API client instance and set it into the
// gin.Context
func AttachApiClient(client remote.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("api_client", client)
		c.Next()
	}
}

// AttachResourceManager attaches the resource manager instance and set it into
// the gin.Context
func AttachResourceManager(m *resource.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("resource_manager", m)
		c.Next()
	}
}

// AttachUserManager attaches the user manager instance and set it into the
// gin.Context
func AttachUserManager(m *user.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_manager", m)
		c.Next()
	}
}

// AttachServerManager attaches the server manager instance and set it into the
// gin.Context
func AttachServerManager(m *server.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("server_manager", m)
		c.Next()
	}
}

// AttachApiLoginKeyManager attaches the API login key manager instance and set
// it into the gin.Context
func AttachApiLoginKeyManager(m *api_login_key.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("api_login_key_manager", m)
		c.Next()
	}
}

// ExtractApiKeyManager returns the API key manager instance and set it into the
// gin.Context
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

// ExtractApiLoginKeyManager returns the api login key manager instance and set it into the
// gin.Context
func ExtractApiLoginKeyManager(c *gin.Context) *api_login_key.Manager {
	if v, ok := c.Get("api_login_key_manager"); ok {
		return v.(*api_login_key.Manager)
	}
	panic("router/middleware: api login key manager not present in context")
}

func ExtractClientManager(c *gin.Context) *client.Manager {
	if v, ok := c.Get("client_manager"); ok {
		return v.(*client.Manager)
	}
	panic("router/middleware: client manager not present in context")
}

// ResourceExists will ensure that the request resource exists in the cache.
// Returns a 404 if it can't be located. If the resource is found it is set into
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
// Returns a 404 if it can't be located it. If the server is found it is set into
// the request context.
func ServerExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param("server") != "" {
			var s *domain.Server
			manager := ExtractServerManager(c)
			server_id := c.Param("server")
			s, err := manager.FindByID(server_id)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found."})
				return
			}
			c.Set("server", s)
		}
		c.Next()
	}
}

// ApiKeyExists loads the API key identified by the :id URL parameter into
// the request context under the key "apiKey". Aborts 404 if not found, 400
// if the parameter is not a valid integer. Use as the loader before any
// RequireCan(action, "apiKey") gate.
func ApiKeyExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.Param("id")
		if raw == "" {
			c.Next()
			return
		}
		id, err := strconv.Atoi(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "API key id must be an integer."})
			return
		}
		k, err := ExtractApiKeyManager(c).FindByID(id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "The requested resource could not be found."})
			return
		}
		c.Set("apiKey", k)
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

