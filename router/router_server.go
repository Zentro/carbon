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
	"carbon/domain"
	"carbon/internal/server"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID   int    `json:"uid"`
	ServerID string `json:"sid"`
	jwt.RegisteredClaims
}

// getAllServers godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	[]domain.Server
//	@Failure	400	{object}	RequestError
//	@Failure	404	{object}	RequestError
//	@Failure	500	{object}	RequestError
//	@Router		/servers [get]
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
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	path		string	true	"Server Identifier"
//	@Success	200		{object}	domain.Server
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server} [get]
func getServer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": ExtractServer(c),
	})
}

// postCreateServer godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	body		domain.Server	true	"Server Object"
//	@Success	200		{object}	domain.Server
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers [post]
func postCreateServer(c *gin.Context) {
	apiKey := ExtractApiKey(c)

	var server domain.Server
	if err := c.BindJSON(&server); err != nil {
		return
	}

	server.ApiKeyID = &apiKey.ApiKeyID

	manager := ExtractServerManager(c)
	if err := manager.Create(&server); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// putUpdateServer godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	path	string			true	"Server Identifier"
//	@Param		server	body	domain.Server	true	"Server Object"
//	@Success	204		"No Content"
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server} [put]
func putUpdateServer(c *gin.Context) {
	var updateServerRequest domain.Server
	if err := c.BindJSON(&updateServerRequest); err != nil {
		return
	}

	manager := ExtractServerManager(c)
	if err := manager.Update(&updateServerRequest); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// patchServerPower godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	path	string	true	"Server Identifier"
//	@Param		data	body	object	true	"Data Object"
//	@Success	204		"No Content"
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server}/power [patch]
func patchServerPower(c *gin.Context) {
	s := ExtractServer(c)
	manager := ExtractServerManager(c)

	var data struct {
		PowerStatus ServerStatus `json:"power_status"`
	}

	if err := c.BindJSON(&data); err != nil {
		return
	}

	// Generally, the server could report itself as "crashed", but that
	// isn't common. Rather, we expect the server to always report itself
	// as "online" or "offline".
	if !data.PowerStatus.IsValid() {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error": "The power status provided was not valid, should be one of \"online\", \"offline\"",
		})
	}

	// Avoid wasting resources by trying to set the server power status
	// to what it already is.
	if s.GetPowerStatus() == data.PowerStatus {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot set the status of the server with the same status.",
		})
	}

	// If we're attempting to set the server "online" we need to verify
	// that we can connect to it. The process should expire before the
	// HTTP connection does so we can return to the requester whether
	// the server power status change was successful or if we couldn't
	// establish whether or not the server is alive.
	if data.PowerStatus.IsOnline() {
		if _, err := server.Connect(s.Host, s.Port, s.Version); err != nil {
			NewError(err).Abort(c)
			return
		}
	}

	s.SetPowerStatus(data.PowerStatus)
	if err := manager.Update(s); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// putSyncServer godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	path	string	true	"Server Identifier"
//	@Success	204		"No Content"
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server}/sync [put]
func putSyncServer(c *gin.Context) {
	s := ExtractServer(c)

	// We only allow a server to in an "online" power status, because otherwise
	// the cleanup routine will continue to treat the server as available when
	// it's really not.
	if !s.GetPowerStatus().IsOnline() {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "Cannot sync a server that is stopped or crashed.",
		})
	}

	if err := ExtractServerManager(c).UpdateLastSync(s.ServerID.String()); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// postCreateServerClient godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	path	string	true	"Server Identifier"
//	@Success	204		"No Content"
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server}/clients [post]
func postCreateServerClient(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// getAllServerClients godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	path	string	true	"Server Identifier"
//	@Success	204		"No Content"
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server}/clients [get]
func getAllServerClients(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// getServerClient godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Param		server	path	string	true	"Server Identifier"
//	@Param		client	path	string	true	"Client Identifier"
//	@Success	204		"No Content"
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server}/clients/{client} [get]
func getServerClient(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func getServerMe(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}
