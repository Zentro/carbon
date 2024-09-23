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
	"carbon/internal/resource"
	"carbon/internal/server"
	"carbon/internal/token"
	"carbon/internal/user"
	"carbon/remote"
	"carbon/system"
	"net/http"

	_ "carbon/docs" // This imports the docs package created by Swag CLI

	"github.com/apex/log"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type ManagerGroup struct {
	ResourceManager *resource.Manager
	ServerManager   *server.Manager
	UserManager     *user.Manager
	TokenManager    *token.Manager
}

func NewClient(remote remote.Client, managers ManagerGroup) *gin.Engine {
	debug := config.Get().Debug
	gin.SetMode(map[bool]string{true: gin.DebugMode, false: gin.ReleaseMode}[debug])

	router := gin.New()

	// If running behind an NGINX proxy.
	router.SetTrustedProxies([]string{"127.0.0.1", "192.168.1.2", "10.0.0.0/8"})

	router.Use(gin.Recovery())
	router.Use(AttachApiClient(remote))
	router.Use(AttachResourceManager(managers.ResourceManager),
		AttachUserManager(managers.UserManager),
		AttachServerManager(managers.ServerManager),
		AttachTokenManager(managers.TokenManager))
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(gin.LoggerWithFormatter(func(params gin.LogFormatterParams) string {
		log.WithFields(log.Fields{
			"client_ip":   params.ClientIP,
			"user_agent":  params.Request.UserAgent(),
			"latency":     params.Latency,
			"status_code": params.StatusCode}).Infof("%s %s", params.MethodColor()+params.Method+params.ResetColor(), params.Path)

		return ""
	}))

	router.Use(AttachCorsHeaders())
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "The requested endpoint could not be found."})
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": system.Version,
		})
	})

	if debug {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	auth := router.Group("/auth")
	auth.POST("/login", postAuthLogin)
	auth.POST("/logout", RequireAuthorization(), postAuthLogout)
	auth.POST("/refresh", postAuthRefresh)

	router.GET("/users/me", RequireAuthorization(), getMe)
	router.GET("/users/:user", getUser)

	router.GET("/servers", getAllServers)
	router.GET("/servers/:server", getServer)

	server := router.Group("/servers/:server")
	server.Use(RequireAuthorization())
	{
		server.POST("", postCreateServer)
		server.PUT("", putUpdateServer)
		server.POST("/sync", postSyncServer)
		server.POST("/power", postServerPower)
	}

	router.GET("/resources", getAllResources)
	router.GET("/resources/:resource", ResourceExists(), getResource)
	router.GET("/resources/:resource/reviews", ResourceExists(), getResourceReviews)
	router.GET("/resources/:resource/versions", ResourceExists(), getResourceVersions)
	router.GET("/resources/:resource/updates", ResourceExists(), getResourceUpdates)
	router.GET("/resource-categories", getAllCategories)
	router.GET("/resource-categories/:category", getCategory)
	router.GET("/resource-versions/:version", getResourceVersion)

	return router
}
