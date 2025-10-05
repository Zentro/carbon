// Copyright (C) 2022 Rafael Galvan <rafael.galvan@rigsofrods.org>

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
	"carbon/config"
	"carbon/domain"
	"carbon/remote"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// UserAuthSessionChallengeClaims represents the high level definition for the
// claims associated with a authentication session challenge.
type UserAuthSessionChallengeClaims struct {
	// The user ID associated with the challenge.
	UserID int `json:"uid"`
	// The server ID associated with the challenge.
	ServerID string `json:"sid"`
	// The typical JWT claims.
	jwt.RegisteredClaims
}

// UserAuthRequest The low level definition for a typical log in request.
type UserAuthRequest struct {
	Login    string `json:"login" form:"login" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
	// The IP that should be considered to be making the request. This will
	// be used to prevent brute force attempts.
	//LimitIp string `json:"limit_ip" form:"limit_ip" binding:"required"`

	TfaProvider string `json:"tfa_provider,omitempty" form:"tfa_provider" binding:"omitempty"`
	TfaTrigger  bool   `json:"tfa_trigger,omitempty" form:"tfa_trigger" binding:"omitempty"`
	Code        string `json:"code,omitempty" form:"code" binding:"omitempty"`

	// Optional but usually always present. If true, the refresh token
	// will have an expiry date of 30 days on the day of issuance.
	// Otherwise, we expire the refresh token after 1 day.
	//Remember bool `json:"remember,omitempty" binding:"omitempty"`

	// The device identification string not to be confused with a literal
	// device ID.
	Device string `json:"device" form:"device" binding:"omitempty"`
}

type UserAuthRefreshRequest struct {
	LoginToken   string `json:"login_token" binding:"required"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// postAuthLogin godoc
//
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	remote.RawUserAuthResponse
//	@Failure	400	{object}	RequestError
//	@Failure	404	{object}	RequestError
//	@Failure	500	{object}	RequestError
//	@Router		/auth/login/ [post]
func postAuthLogin(c *gin.Context) {
	var authRequest UserAuthRequest
	var authResponse remote.RawUserAuthResponse
	if err := c.BindJSON(&authRequest); err != nil {
		return
	}

	authResponse, err := ExtractApiClient(c).ValidateUserAuthCredentials(c, authRequest)
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if authResponse.TfaRequired {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"tfa_providers": authResponse.TfaProviders,
		})
		return
	}

	if authResponse.TfaTriggered {
		c.AbortWithStatusJSON(http.StatusAccepted, gin.H{
			"tfa_triggered": authResponse.TfaTriggered,
		})
		return
	}

	user := authResponse.User

	manager := ExtractApiLoginKeyManager(c)
	token, err := domain.NewApiLoginKey(user.UserID, c.ClientIP())
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if err := manager.Create(token); err != nil {
		NewError(err).Abort(c)
		return
	}

	authResponse.LoginToken = token.LoginKey
	authResponse.RefreshToken = token.RefreshKey

	c.JSON(http.StatusOK, authResponse)
}

// postAuthLogout godoc
//
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	remote.RawUserAuthResponse
//	@Failure	400	{object}	RequestError
//	@Failure	404	{object}	RequestError
//	@Failure	500	{object}	RequestError
//	@Router		/auth/logout/ [post]
func postAuthLogout(c *gin.Context) {
	if err := ExtractApiLoginKeyManager(c).Invalidate(ExtractApiLoginKey(c).ApiLoginKeyID); err != nil {
		NewError(err).Abort(c)
		return
	}
	c.Status(http.StatusNoContent)
}

// postAuthRefresh godoc
//
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	remote.RawUserAuthResponse
//	@Failure	400	{object}	RequestError
//	@Failure	404	{object}	RequestError
//	@Failure	500	{object}	RequestError
//	@Router		/auth/refresh/ [post]
func postAuthRefresh(c *gin.Context) {
	var authRefreshRequest UserAuthRefreshRequest
	if err := c.BindJSON(&authRefreshRequest); err != nil {
		return
	}

	var apiKey domain.ApiLoginKey

	manager := ExtractApiLoginKeyManager(c)
	apiKey, err := manager.FindByToken(authRefreshRequest.LoginToken)
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if c.ClientIP() != apiKey.IP {
		NewError(ErrIpMismatch).Abort(c)
		return
	}

	if time.Now().After(apiKey.RefreshKeyExpiresAt) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "The authorization heads are expired and can not be reissued.",
		})
		return
	}

	newApiKey, err := domain.NewApiLoginKey(apiKey.UserID, apiKey.IP)
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if err := manager.Refresh(newApiKey); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"login_token":   newApiKey.LoginKey,
		"refresh_token": newApiKey.RefreshKey,
	})
}

// postAuthSessionsJoin godoc
//
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		data	body	object	true	"Data Object"
//	@Success	200		{object}	string
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/auth/sessions/join [post]
func postAuthSessionsJoin(c *gin.Context) {
	user := ExtractUser(c)
	manager := ExtractServerManager(c)

	var data struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := manager.FindByHostAndPort(data.Host, data.Port)
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	// The player wants his client to start a chllange so the server
	// can verify their session against us. The challenge should only
	// last 10 seconds, and we should verify his claims against ours.
	claims := UserAuthSessionChallengeClaims{
		UserID:   user.UserID,
		ServerID: server.ServerID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	challenge := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedChallenge, err := challenge.SignedString([]byte(config.Get().Secret))
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	c.String(http.StatusOK, signedChallenge)
}

// getAuthSessionsVerify godoc
//
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		server	path	string	true	"Server Identifier"
//	@Success	204		"No Content"
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/auth/session/{server}/verify [get]
func getAuthSessionsVerify(c *gin.Context) {
	server := ExtractServer(c)

	var data struct {
		Challenge string `json:"challenge"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println(data.Challenge)

	claims := &UserAuthSessionChallengeClaims{}
	challenge, err := jwt.ParseWithClaims(data.Challenge, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Get().Secret), nil
	})

	if err != nil {
		fmt.Println(err.Error())
		NewError(err).Abort(c)
		return
	}

	if !challenge.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "The provided challenge could not be validated.",
		})
		return
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "The provided challenge has already expired.",
		})
		return
	}

	// Make sure we don't allow a potential clash when a server may
	// attempt to process a challenge for another server.
	if server.ServerID.String() != claims.ServerID {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "The provided challenge has claims that could not be validated.",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
