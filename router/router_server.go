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
	"net/http"

	"github.com/gin-gonic/gin"
)

func getAllServers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"servers": ExtractServerManager(c).Collection(),
	})
}

func getServer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": ExtractServer(c),
	})
}

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

func putUpdateServer(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func postServerPower(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func postSyncServer(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func postCreateServerPlayer(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func getAllServerPlayers(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func getServerPlayer(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}
