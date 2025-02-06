// Copyright (C) 2022 Rafael Galvan

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
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID   int `json:"uid"`
	ServerID int `json:"sid"`
	jwt.RegisteredClaims
}

// getAllServers godoc
// @Tags         server
// @Accept       json
// @Produce      json
// @Success      200  {object}  []domain.Server
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /servers [get]
func getAllServers(c *gin.Context) {
	servers, err := ExtractServerManager(c).Collection()
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": servers,
	})
}

// getServer godoc
// @Tags         server
// @Accept       json
// @Produce      json
// @Success      200  {object}  domain.Server
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /servers/{server} [get]
func getServer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": ExtractServer(c),
	})
}

// postCreateServer godoc
// @Tags         server
// @Accept       json
// @Produce      json
// @Success      200  {object}  domain.Server
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /servers [post]
func postCreateServer(c *gin.Context) {
	var newServerRequest domain.Server
	if err := c.BindJSON(&newServerRequest); err != nil {
		return
	}

	manager := ExtractServerManager(c)
	if err := manager.Create(&newServerRequest); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// putUpdateServer godoc
// @Tags         server
// @Accept       json
// @Produce      json
// @Success      200  {object}  domain.Server
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /servers [put]
func putUpdateServer(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func postServerPower(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func postSyncServer(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func postCreateServerClient(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func getAllServerClients(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func getServerClient(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func postClientJoinRequest(c *gin.Context) {
	user := ExtractUser(c)
	server := ExtractServer(c)

	claims := CustomClaims{
		UserID:   user.UserID,
		ServerID: server.ServerID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(60 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.Get().Secret))
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	c.String(http.StatusOK, signedToken)
}

func getClientJoinRequest(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}

	if err := c.BindJSON(&req); err != nil {
		return
	}

	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(req.Token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Get().Secret), nil
	})

	if err != nil {
		NewError(err).Abort(c)
		return
	}

	if !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "The provided token could not be validated.",
		})
		return
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "The provided token has already expired.",
		})
		return
	}

	c.Status(http.StatusAccepted)
}
