// Copyright (C) 2025 Rafael Galvan <rafael.galvan@rigsofrods.org>

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

package router

import (
	"errors"
	"net/http"

	"carbon/domain"
	"carbon/internal/policy"

	"github.com/gin-gonic/gin"
)

const authorizerCtxKey = "authorizer"

// AttachAuthorizer wires a single shared *policy.Authorizer into every
// request context. Mount this once at the root of the router.
func AttachAuthorizer(a *policy.Authorizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(authorizerCtxKey, a)
		c.Next()
	}
}

// ExtractAuthorizer returns the request's authorizer. Panics if the
// AttachAuthorizer middleware was not mounted.
func ExtractAuthorizer(c *gin.Context) *policy.Authorizer {
	v, ok := c.Get(authorizerCtxKey)
	if !ok {
		panic("router/middleware_policy: authorizer not present in context")
	}
	return v.(*policy.Authorizer)
}

// RequireCan gates a handler by calling Authorizer.Can. The targetCtxKey
// names a previously-loaded entity (set by an Exists-style middleware) that
// must implement domain.Manageable; pass "" for collection-level actions.
//
//	serverGrp.GET("",          RequireCan(domain.ActionServerList,  ""))
//	oneServer.GET("",          RequireCan(domain.ActionServerRead,  "server"))
//	oneServer.PATCH("",        RequireCan(domain.ActionServerWrite, "server"))
//
// Must be mounted *after* RequireAuth and (for non-empty targetCtxKey) the
// loader middleware that places the target into the context.
func RequireCan(action domain.Action, targetCtxKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := ExtractPrincipal(c)
		authz := ExtractAuthorizer(c)

		var target domain.Manageable
		if targetCtxKey != "" {
			v, ok := c.Get(targetCtxKey)
			if !ok {
				// Programmer error: the route declared a target key but the
				// loader middleware didn't run or set the wrong key. Better
				// to panic loudly than silently authorize.
				panic("router/middleware_policy: target ctx key " + targetCtxKey + " not present; loader middleware missing?")
			}
			t, ok := v.(domain.Manageable)
			if !ok {
				panic("router/middleware_policy: target at ctx key " + targetCtxKey + " does not implement domain.Manageable")
			}
			target = t
		}

		if err := authz.Can(c, p, action, target); err != nil {
			status, msg := translatePolicyError(err)
			c.AbortWithStatusJSON(status, gin.H{"error": msg})
			return
		}
		c.Next()
	}
}

func translatePolicyError(err error) (int, string) {
	switch {
	case errors.Is(err, policy.ErrInsufficientScope):
		return http.StatusForbidden, "Your credentials do not grant the required scope for this action."
	case errors.Is(err, policy.ErrForbidden):
		return http.StatusForbidden, "You are not authorized to access this resource."
	case errors.Is(err, policy.ErrKindMismatch):
		// This indicates a route-wiring bug, not an auth failure. Surface it
		// as 500 so it shows up in error logs rather than being mistaken for
		// a normal 403.
		return http.StatusInternalServerError, "Internal authorization configuration error."
	default:
		return http.StatusForbidden, "You are not authorized."
	}
}
