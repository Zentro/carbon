package router

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ShowAccount godoc
// @Tags         resource
// @Accept       json
// @Produce      json
// @Success      200  {object}  []domain.Resource
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /resources/ [get]
func getAllResources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"resources": ExtractResourceManager(c).Collection(),
	})
}

// ShowAccount godoc
// @Tags         resource
// @Accept       json
// @Produce      json
// @Success      200  {object}  domain.Resource
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /resources/{resource} [get]
func getResource(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"resource": ExtractResource(c),
	})
}

// ShowAccount godoc
// @Tags         resource
// @Accept       json
// @Produce      json
// @Success      200  {object}  []domain.ResourceReview
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /resources/{resource}/reviews [get]
func getResourceReviews(c *gin.Context) {
	client := ExtractApiClient(c)
	r := ExtractResource(c)

	reviews, err := client.GetResourceReviews(c, r.ID())
	if err != nil {
		NewError(err).Abort(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"reviews": reviews,
	})
}

// ShowAccount godoc
// @Tags         resource
// @Accept       json
// @Produce      json
// @Success      200  {object}  []domain.ResourceVersion
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /resources/{resource}/versions [get]
func getResourceVersions(c *gin.Context) {
	client := ExtractApiClient(c)
	r := ExtractResource(c)

	versions, err := client.GetResourceVersions(c, r.ID())
	if err != nil {
		NewError(err).Abort(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
	})
}

// ShowAccount godoc
// @Tags         resource
// @Accept       json
// @Produce      json
// @Success      200  {object}  domain.ResourceVersion
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /resource-versions/{version} [get]
func getResourceVersion(c *gin.Context) {
	version, err := ExtractApiClient(c).GetResourceVersion(c, c.Param("version"))
	if err != nil {
		NewError(err).Abort(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"version": version,
	})
}

func getResourceUpdates(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

// ShowAccount godoc
// @Tags         resource
// @Accept       json
// @Produce      json
// @Success      200  {object}  []domain.ResourceCategory
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /resource-categories/ [get]
func getAllCategories(c *gin.Context) {
	res, _, err := ExtractApiClient(c).GetResourceCategories(c)
	if err != nil {
		fmt.Println(err.Error())
		NewError(err).Abort(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"categories": res,
	})
}

// ShowAccount godoc
// @Tags         resource
// @Accept       json
// @Produce      json
// @Success      200  {object}  domain.ResourceCategory
// @Failure      400  {object}  RequestError
// @Failure      404  {object}  RequestError
// @Failure      500  {object}  RequestError
// @Router       /resource-categories/{category} [get]
func getCategory(c *gin.Context) {
	res, err := ExtractApiClient(c).GetResourceCategory(c)
	if err != nil {
		NewError(err).Abort(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"category": res,
	})
}
