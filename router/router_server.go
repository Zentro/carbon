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
	"carbon/internal/api_key"
	"carbon/internal/server"
	"net/http"
	"strconv"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

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
	p := ExtractPrincipal(c)
	servers := ExtractServerManager(c)
	keys := ExtractApiKeyManager(c)

	// A bound credential cannot create new servers — its authority is
	// scoped to one entity. Unbound personal keys and login keys can.
	if p.IsBound() {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Bound credentials cannot create servers; use an unbound key or sign in.",
		})
		return
	}

	var srv domain.Server
	if err := c.BindJSON(&srv); err != nil {
		return
	}
	srv.OwnerUserID = p.UserID

	if err := servers.Create(&srv); err != nil {
		NewError(err).Abort(c)
		return
	}

	// Generate the per-server bound API key the game server will use to
	// authenticate. Returned exactly once in the response body.
	plaintext, err := api_key.GenerateRandomKey()
	if err != nil {
		NewError(err).Abort(c)
		return
	}
	kind := srv.Kind()
	id := srv.EntityID()
	bound := &domain.ApiKey{
		UserID:     p.UserID,
		Name:       "Server: " + srv.Name,
		Key:        plaintext,
		Scopes:     domain.NewScopeSet(string(domain.ActionServerRead), string(domain.ActionServerWrite)),
		TargetKind: &kind,
		TargetID:   &id,
		Enabled:    true,
	}
	if err := keys.Create(bound); err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server":  srv,
		"api_key": plaintext,
	})
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
	var server domain.Server
	if err := c.BindJSON(&server); err != nil {
		return
	}

	manager := ExtractServerManager(c)
	if err := manager.Update(&server); err != nil {
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

	// Only the values defined in the ServerStatus type are valid.
	// The power status can really only ever be defined by us.
	// The user can set it to "online" only after we verify that we can
	// actually connect to the server.
	// The user can set it to "offline" at any time.
	// Any other value is invalid.
	if !data.PowerStatus.IsValid() {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error": "The power status provided was not valid, should be one of \"online\", \"offline\"",
		})
	}

	// If the power status is already set to the requested value, do nothing.
	if s.GetPowerStatus() == data.PowerStatus {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The power status is already set to the requested value",
		})
	}

	// If we're attempting to set the server "online" we need to verify
	// that we can actually connect to it first. This is a BLOCKING call.
	if data.PowerStatus.IsOnline() {
		if err := server.Knock(*s, requestid.Get(c)); err != nil {
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
			"error": "Cannot sync a server that is one of \"stopped\", \"crashed\"",
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
	var client domain.Client
	if err := c.BindJSON(&client); err != nil {
		return
	}

	m := ExtractClientManager(c)
	server := ExtractServer(c)
	m.Create(&client, server)

	c.JSON(http.StatusOK, gin.H{
		"client": client,
	})
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
	m := ExtractClientManager(c)
	server := ExtractServer(c)

	clients, err := m.CollectionByServerID(server.ServerID.String())
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clients": clients,
	})
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
	m := ExtractClientManager(c)
	id, err := strconv.Atoi(c.Param("client"))
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	client, err := m.FindByID(id)
	if err != nil {
		NewError(err).Abort(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"client": client,
	})
}
