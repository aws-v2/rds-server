package http

import (
	"rds/internal/application"

	"github.com/gin-gonic/gin"
)


type DocsHandler struct {
	service *application.DocsService
}
func NewDocsHandler(service *application.DocsService) *DocsHandler {
	return &DocsHandler{service: service}
}
func (h *DocsHandler) GetManifest(c *gin.Context) {
	role := c.GetString("role")
	if role == "USER" {
		data, err := h.service.GetManifest(false)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"data": data})
		return
	}

	// role is anything else: serves both public and private/internal manifest categories merged
	pubData, err := h.service.GetManifest(false)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	intData, err := h.service.GetManifest(true)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Merge categories
	mergedCategories := append(pubData.Categories, intData.Categories...)
	combined := &application.DocManifest{
		Service:    pubData.Service,
		Version:    pubData.Version,
		Categories: mergedCategories,
	}

	c.JSON(200, gin.H{"data": combined})
}

func (h *DocsHandler) GetDoc(c *gin.Context) {
	slug := c.Param("slug")
	role := c.GetString("role")

	if role == "USER" {
		doc, err := h.service.GetDoc(slug, false)
		if err != nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		c.JSON(200, gin.H{"data": doc})
		return
	}

	// Try public first
	doc, err := h.service.GetDoc(slug, false)
	if err != nil {
		// Try internal
		doc, err = h.service.GetDoc(slug, true)
		if err != nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
	}

	c.JSON(200, gin.H{"data": doc})
}