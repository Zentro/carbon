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

import "strconv"

type Resource struct {
	ResourceId         int         `json:"resource_id"`
	ResourceCategoryId uint        `json:"resource_category_id"`
	ResourceState      string      `json:"resource_state"`
	ResourceType       string      `json:"resource_type"`
	Title              string      `json:"title"`
	TagLine            string      `json:"tag_line"`
	UpdateCount        int         `json:"update_count"`
	ExternalUrl        string      `json:"external_url,omitempty"`
	ViewUrl            string      `json:"view_url"`
	IconUrl            string      `json:"icon_url"`
	CurrentDownloadUrl string      `json:"current_download_url,omitempty"`
	ViewCount          uint        `json:"view_count"`
	ReviewCount        int         `json:"review_count"`
	DownloadCount      uint        `json:"download_count"`
	RatingCount        uint        `json:"rating_count"`
	RatingAvg          float64     `json:"rating_avg"`
	RatingWeighted     float64     `json:"rating_weighted"`
	LastUpdate         uint        `json:"last_update"`
	ResourceDate       uint        `json:"resource_date"`
	Version            string      `json:"version"`
	License            string      `json:"license,omitempty"`
	LicenseUrl         string      `json:"license_url,omitempty"`
	Description        string      `json:"description,omitempty"`
	CustomFields       interface{} `json:"custom_fields,omitempty"`
	CanDownload        bool        `json:"can_download"`
}

func (r *Resource) ID() string {
	return strconv.Itoa(r.ResourceId)
}

type ResourceReview struct {
	Message           string `json:"message,omitempty"`
	ResourceRatingId  uint   `json:"resource_rating_id"`
	Rating            uint   `json:"rating"`
	RatingDate        uint   `json:"rating_date"`
	RatingState       string `json:"rating_state"`
	ResourceVersionId uint   `json:"resource_version_id"`
	ResourceId        uint   `json:"resource_id"`
	//User              user.User `json:"User"`
}

type ResourceCategory struct {
	ResourceCategoryId int    `json:"resource_category_id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	ResourceCount      uint   `json:"resource_count"`
	LastUpdate         uint   `json:"last_update"`
	ParentCategoryId   uint   `json:"parent_category_id"`
	DisplayOrder       uint   `json:"display_order"`
}

func (rc *ResourceCategory) ID() string {
	return strconv.Itoa(rc.ResourceCategoryId)
}

type ResourceVersion struct {
	ResourceVersionId uint           `json:"resource_version_id"`
	ResourceId        uint           `json:"resource_id"`
	ReleaseDate       uint           `json:"release_date"`
	Files             []ResourceFile `json:"files"`
	VersionString     string         `json:"version_string"`
	DownloadCount     uint           `json:"download_count"`
}

type ResourceUpdate struct {
	ResourceUpdateId int    `json:"resource_update_id"`
	ResourceId       int    `json:"resource_id"`
	Message          string `json:"message"`
	Title            string `json:"title"`
	ViewUrl          string `json:"view_url"`
	PostDate         uint   `json:"post_Date"`
	AttachCount      int    `json:"attach_count"`
}

type ResourceFile struct {
	Id       uint   `json:"id"`
	FileName string `json:"filename"`
	Size     uint   `json:"size"`
	// DownloadUrl string `json:"download_url"`
}
