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

package main

import (
	"carbon/cmd"
)

// .
// ├── cmd
// │   └── root
// ├── config
// ├── entity
// │   ├── resource
// │   ├── user
// │   └── server
// ├── mysql
// │   └── servers
// ├── remote
// │   ├── errors
// │   ├── http
// │   ├── resources
// │   ├── types
// │   └── users
// ├── router
// │   ├── tokens
// │   │   ├── errors
// │   │   └── store
// │   ├── errors
// │   ├── middleware
// │   ├── router_auth
// │   ├── router_download
// │   ├── router_resource
// │   ├── router_server_ws
// │   ├── router_server
// │   └── router_user
// ├── system
// │   └── const

// @title           Rigs of Rods API
// @version         2.0

// @contact.name   Rafael Galvan
// @contact.url    http://www.rigsofrods.org

// @license.name  GNU GPL v3
// @license.url   https://www.gnu.org/licenses/gpl-3.0.en.html

// @host      localhost:8080
// @BasePath  /

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	cmd.Execute()
}
