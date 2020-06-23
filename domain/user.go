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

package domain

// User represents a high level definition of a user.
type User struct {
	UserID            int         `json:"user_id"`
	UserTitle         string      `json:"user_title"`
	Email             string      `json:"email"`
	Name              string      `json:"username"`
	IsStaff           bool        `json:"is_staff"`
	Gravatar          string      `json:"gravatar,omitempty"`
	AvatarUrls        interface{} `json:"avatar_urls,omitempty"`
	ProfileBannerUrls interface{} `json:"profile_banner_urls,omitempty"`
	ViewUrl           string      `json:"view_url"`
}
