## Introduction

Rigs of Rods API (carbon) is the open source web API designed to interface with XenForo 2.2^ and the XenForo Resource Manager 2.2^ to provide a seamless experience to the Rigs of Rods client and server. It is written in Go.

## Requirements

* Go 1.23
* UPX (not really required, but can be useful)
* MariaDB

## Installation

**NOTE: This is meant to be done on a Linux machine, there are no instructions for Windows.**

### Quick setup

This will install the Carbon executable in the expected directory structure.
```
mkdir -p /etc/carbon
curl -L -o /usr/local/bin/carbon ""
chmod u+x /usr/local/bin/carbon
```

### Compiling from source

It is recommended to **NOT** clone the `develop` branch but instead clone a specific version (preferably the latest) of carbon.
```
git clone -b <version> https://github.com/Zentro/carbon
make build; compress
```

### Configuration

Carbon will read from `/etc/carbon` so you will need to create a `config.yml` and paste the contents from the specific version branch.

You will also see that you will need to have MariaDB installed (`sudo apt install mariadb-server`) and have it properly configured with a user (you can call it `carbon` or whatever). The SQL dump file must be imported to a database (which can also be called `carbon` or whatever).

Carbon logs with logrotate with the expectation that you will create a `/var/log/carbon`. You can change this in the configuration.

### Swaggo

We use swaggo to generate documentation for the API endpoints. **They will only be generated and accessible if in debug mode, otherwise this should be disabled in production.**

### Running or Running as a Daemon

You can run with `carbon` and add the `--debug` flag to run in debug mode.

If you would prefer to run carbon in the background, you will need to create a systemd service `carbon.service` in the `/etc/systemd/system` directory.

```
[Unit]
Description=Carbon Daemon

[Service]
User=root
WorkingDirectory=/etc/carbon
LimitNOFILE=4096
PIDFile=/var/run/carbon/daemon.pid
ExecStart=/usr/local/bin/carbon
Restart=on-failure
StartLimitInterval=180
StartLimitBurst=30
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

You can of course make changes such as the user it runs as.

## Using Docker

You can also use Docker instead if your infrastructure demands it.

```
docker compose up
```

## Project Layout

This is always subject to change. Certain elements of the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) are used such as `cmd`. Other aspects are completely made up to conform to the multiple roles and interfaces Carbon deals with.

```
.
├── cmd
│   └── root
├── config
├── entity
│   ├── resource
│   ├── user
│   └── server
├── mysql
│   └── servers
├── remote
│   ├── errors
│   ├── http
│   ├── resources
│   ├── types
│   └── users
├── router
│   ├── tokens
│   │   ├── errors
│   │   └── store
│   ├── errors
│   ├── middleware
│   ├── router_auth
│   ├── router_download
│   ├── router_resource
│   ├── router_server_ws
│   ├── router_server
│   └── router_user
├── system
│   └── const
```

## License

Copyright (C) 2022-2024 Rafael Galvan <system(at)zentro.codes>

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see <https://www.gnu.org/licenses/>.