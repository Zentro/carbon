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
	"carbon/internal/api_key"
	"carbon/internal/client"
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
	ApiKeyManager   *api_key.Manager
	ClientManager   *client.Manager
}

type Role = domain.ApiKeyRole
type ServerStatus = domain.ServerStatus

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
		AttachTokenManager(managers.TokenManager),
		AttachApiKeyManager(managers.ApiKeyManager))
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

	router.GET("/ip", getClientIP)

	if debug {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	auth := router.Group("/auth")
	auth.POST("/login", postAuthLogin)
	auth.POST("/logout", RequireAuthorization(), postAuthLogout)
	auth.POST("/refresh", postAuthRefresh)
	auth.POST("/sessions/join",
		RequireAuthorization(),
		postAuthSessionsJoin,
	)
	auth.GET("/sessions/:server/verify",
		//RequireApiAuthorization(),
		//RoleRequired(Role("user")),
		ServerExists(),
		//RequireResourceOwnership(),
		getAuthSessionsVerify,
	)

	router.GET("/users/me", RequireAuthorization(), getMe)
	router.GET("/users/:user", getUser)

	router.GET("/servers", getAllServers)
	router.GET("/servers/:server", ServerExists(), getServer)

	router.POST("/servers", RequireApiAuthorization(), RoleRequired(Role("user")), postCreateServer)

	server := router.Group("/servers/:server")
	server.Use(
		RequireApiAuthorization(),
		RoleRequired(Role("user")),
		ServerExists(),
		RequireResourceOwnership(),
	)
	{
		server.PUT("", putUpdateServer)
		server.PUT("/sync", putSyncServer)
		server.PATCH("/power", patchServerPower)

		server.GET("/clients", getAllServerClients)
		server.GET("/clients/:client", getServerClient)
		server.POST("/clients", postCreateServerClient)
	}

	client := router.Group("/clients")
	client.Use(RoleRequired(Role("operator")))
	{
		client.GET("")
		client.GET("/:client")
	}

	api_key := router.Group("/api-keys")
	api_key.Use(RequireApiAuthorization(), RoleRequired(Role("operator")))
	{
		api_key.POST("", postCreateApiKey)
		api_key.GET("/:api_key", getApiKey, ApiKeyKeyExists())
		api_key.GET("", getAllApiKeys)
		api_key.DELETE("/:api_key", deleteApiKey, ApiKeyKeyExists())
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

// getClientIP godoc
//
//	@Tags		server
//	@Accept		json
//	@Produce	json
//	@Success	200		{object}	map
//	@Failure	400		{object}	RequestError
//	@Failure	404		{object}	RequestError
//	@Failure	500		{object}	RequestError
//	@Router		/servers/{server}/sync [put]
func getClientIP(c *gin.Context) {
	ip := c.ClientIP()
	c.JSON(http.StatusOK, gin.H{"ip": ip})
}
