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
	"carbon/domain"
	"carbon/remote"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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

// ShowAccount godoc
// @Summary      Processes user authentication by validating credentials.
// @Description  Validates provided username and password to authenticate the user. No session management is handled.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  remote.RawUserAuthResponse
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /auth/login/ [post]
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

	user := authResponse.User

	manager := ExtractTokenManager(c)
	token, err := domain.NewToken(user.UserID, c.ClientIP())
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if err := manager.Create(token); err != nil {
		NewError(err).Abort(c)
		return
	}

	authResponse.LoginToken = token.LoginToken
	authResponse.RefreshToken = token.RefreshToken

	c.JSON(http.StatusOK, authResponse)
}

// ShowAccount godoc
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  remote.RawUserAuthResponse
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /auth/logout/ [post]
func postAuthLogout(c *gin.Context) {
	if err := ExtractTokenManager(c).Invalidate(ExtractToken(c).ID); err != nil {
		NewError(err).Abort(c)
		return
	}
	c.Status(http.StatusNoContent)
}

// ShowAccount godoc
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  remote.RawUserAuthResponse
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /auth/refresh/ [post]
func postAuthRefresh(c *gin.Context) {
	var authRefreshRequest UserAuthRefreshRequest
	if err := c.BindJSON(&authRefreshRequest); err != nil {
		return
	}

	var token domain.Token

	manager := ExtractTokenManager(c)
	token, err := manager.FindByToken(authRefreshRequest.LoginToken)
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if c.ClientIP() != token.IPAddress {
		NewError(ErrIpMismatch).Abort(c)
		return
	}

	if time.Now().After(token.RefreshTokenExpiresAt) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "The authorization heads are expired and can not be reissued.",
		})
		return
	}

	newToken, err := domain.NewToken(token.UserID, token.IPAddress)
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if err := manager.Refresh(newToken); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"login_token":   newToken.LoginToken,
		"refresh_token": newToken.RefreshToken,
	})
}
