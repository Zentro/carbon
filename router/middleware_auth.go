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
	"net/http"
	"strings"
	"time"

	"carbon/domain"

	"github.com/gin-gonic/gin"
)

const principalCtxKey = "principal"

// RequireAuth authenticates the caller and sets a domain.Principal in the
// request context. It accepts either:
//
//   - a login token via "Authorization: Bearer <login_key>" (browser sessions)
//   - an API key via   "Api-Authorization: Bearer <key>"   (game / machine)
//
// On success, handlers can call ExtractPrincipal(c). On failure the request
// is aborted with 401/403. This is the only auth middleware in the system.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if loginToken, ok := bearerFrom(c, "Authorization"); ok {
			if p, err := authenticateLogin(c, loginToken); err == nil {
				c.Set(principalCtxKey, p)
				c.Next()
				return
			} else {
				abortAuth(c, http.StatusForbidden, err.Error())
				return
			}
		}

		if apiKey, ok := bearerFrom(c, "Api-Authorization"); ok {
			if p, err := authenticateApiKey(c, apiKey); err == nil {
				c.Set(principalCtxKey, p)
				c.Next()
				return
			} else {
				abortAuth(c, http.StatusForbidden, err.Error())
				return
			}
		}

		c.Header("WWW-Authenticate", "Bearer")
		abortAuth(c, http.StatusUnauthorized, "Authentication required.")
	}
}

// ExtractPrincipal returns the authenticated principal or panics if the
// route was not gated by RequireAuth.
func ExtractPrincipal(c *gin.Context) domain.Principal {
	v, ok := c.Get(principalCtxKey)
	if !ok {
		panic("router/middleware_auth: principal not present; route is missing RequireAuth()")
	}
	return v.(domain.Principal)
}

// bearerFrom parses a "Bearer <token>" header value. Returns ("", false) if
// the header is absent or malformed.
func bearerFrom(c *gin.Context, header string) (string, bool) {
	parts := strings.SplitN(c.GetHeader(header), " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func abortAuth(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

// authenticateLogin validates a login token and builds a Principal. The
// user record (and therefore the user's role) is fetched from XF on every
// request — cache later if traffic warrants it.
func authenticateLogin(c *gin.Context, token string) (domain.Principal, error) {
	manager := ExtractApiLoginKeyManager(c)
	row, err := manager.FindByToken(token)
	if err != nil {
		return domain.Principal{}, errAuth("Invalid or expired session.")
	}
	if time.Now().After(row.LoginKeyExpiresAt) {
		return domain.Principal{}, errAuth("Session expired.")
	}

	user, httpErr := ExtractApiClient(c).GetUser(c, row.UserID)
	if httpErr != nil {
		return domain.Principal{}, errAuth("Could not load user for session.")
	}

	role := domain.RoleOf(user)
	return domain.Principal{
		UserID:         user.UserID,
		User:           user,
		CredentialKind: domain.CredentialLogin,
		CredentialID:   int(row.ApiLoginKeyID),
		Role:           role,
		Scopes:         role.Scopes(),
		IP:             c.ClientIP(),
	}, nil
}

// authenticateApiKey validates an API key and builds a Principal. The
// caller's role is taken from the XF user record (IsStaff -> admin),
// effective scopes are user.Role.Scopes() ∩ key.Scopes, and the principal
// inherits any binding the key carries.
//
// TODO(perf): this round-trips XF on every machine request. Cache user
// records in-process with a short TTL once traffic warrants it.
func authenticateApiKey(c *gin.Context, key string) (domain.Principal, error) {
	manager := ExtractApiKeyManager(c)
	row, err := manager.FindByKey(key)
	if err != nil || !row.Enabled {
		return domain.Principal{}, errAuth("Invalid or disabled API key.")
	}

	user, httpErr := ExtractApiClient(c).GetUser(c, row.UserID)
	if httpErr != nil {
		return domain.Principal{}, errAuth("Could not load user for API key.")
	}

	role := domain.RoleOf(user)
	effective := role.Scopes()
	if !row.Scopes.Empty() {
		effective = role.Scopes().Intersect(row.Scopes)
	}

	p := domain.Principal{
		UserID:         user.UserID,
		User:           user,
		CredentialKind: domain.CredentialApi,
		CredentialID:   row.ApiKeyID,
		Role:           role,
		Scopes:         effective,
		IP:             c.ClientIP(),
	}
	if row.IsBound() {
		p.BoundKind = *row.TargetKind
		p.BoundID = *row.TargetID
	}
	return p, nil
}

type authError string

func (e authError) Error() string { return string(e) }

func errAuth(msg string) error { return authError(msg) }
