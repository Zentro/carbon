// Copyright (C) 2022-2023 Rafael Galvan

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

package config

import (
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

const DefaultLocation = "/etc/carbon/config.yml"

var (
	mu      sync.RWMutex
	_config *Configuration
)

type Configuration struct {
	path string

	Key string `yaml:"key"`

	Secret string `yaml:"secret"`

	LogDirectory string `default:"/var/log/carbon" yaml:"log_directory"`

	RootDirectory string `default:"/var/lib/carbon" yaml:"root_directory"`

	// Should run in debug mode or production mode. This value is ignored
	// if the debug flag is passed in command line arguments.
	Debug bool `default:"true" yaml:"debug"`

	Api    ApiConfiguration    `yaml:"api"`
	Db     DbConfiguration     `yaml:"db"`
	Remote RemoteConfiguration `yaml:"remote"`
}

type RemoteConfiguration struct {
	Location string `yaml:"location"`
	Key      string `yaml:"key"`
}

type ApiConfiguration struct {
	Host string `default:"0.0.0.0" yaml:"host"`
	Port int    `default:"8080" yaml:"port"`

	Ssl struct {
		Enabled         bool   `default:"true" yaml:"enabled"`
		CertificateFile string `default:"" yaml:"certifcate_file"`
		KeyFile         string `default:"" yaml:"key_file"`
	}

	ReadTimeout  time.Duration `default:"15" yaml:"read_timeout"`
	WriteTimeout time.Duration `default:"15" yaml:"write_timeout"`
	IdleTimeout  time.Duration `default:"60" yaml:"idle_timeout"`
}

type DbConfiguration struct {
	Host string `default:"0.0.0.0" yaml:"host"`
	Port int    `default:"3306" yaml:"port"`

	Database string `default:"" yaml:"database"`
	Username string `default:"" yaml:"username"`
	Password string `default:"" yaml:"password"`

	Charset   string `default:"utf8" yaml:"charset"`
	Collation string `default:"utf8_unicode_ci" yaml:"collation"`
}

func FromFile(path string) error {
	f, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	c := new(Configuration)
	c.path = path

	if err := yaml.Unmarshal(f, c); err != nil {
		return err
	}

	Set(c)
	return nil
}

func Set(c *Configuration) {
	mu.Lock()
	_config = c
	mu.Unlock()
}

func Get() *Configuration {
	mu.RLock()
	c := *_config
	mu.RUnlock()
	return &c
}
