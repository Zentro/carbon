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

package remote

import "carbon/domain"

type d map[string]interface{}

type q map[string]string

type TreeMap map[string]uint

type Pagination struct {
	CurrentPage uint `json:"current_page"`
	LastPage    uint `json:"last_page"`
	PerPage     uint `json:"per_page"`
	Shown       uint `json:"shown"`
	Total       uint `json:"total"`
}

type RawUserAuthResponse struct {
	TfaRequired  bool        `json:"tfa_required,omitempty"`
	TfaProviders string      `json:"tfa_providers,omitempty"`
	TfaTriggered bool        `json:"tfa_triggered,omitempty"`
	User         domain.User `json:"user,omitempty"`
	LoginToken   string      `json:"login_token,omitempty"`
	RefreshToken string      `json:"refresh_token,omitempty"`
}
