package http

import (
	"net/http"
	"rds/internal/application"

	"github.com/gin-gonic/gin"
)

// DocsHandler handles requests for system documentation
type DocsHandler struct {
	docsService *application.DocsService
}

// NewDocsHandler creates a new documentation handler
func NewDocsHandler(docsService *application.DocsService) *DocsHandler {
	return &DocsHandler{
		docsService: docsService,
	}
}

// GetManifest returns the structured table of contents for documentation
func (h *DocsHandler) GetManifest(c *gin.Context) {
	manifest, err := h.docsService.GetManifest(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, "failed to load documentation manifest", nil)
		return
	}

	respond(c, http.StatusOK, "Documentation manifest fetched successfully", manifest)
}

// GetDocContent returns the raw markdown content for a specific documentation page
func (h *DocsHandler) GetDocContent(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		respond(c, http.StatusBadRequest, "slug is required", nil)
		return
	}

	content, err := h.docsService.GetDocContent(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "documentation not found" {
			respond(c, http.StatusNotFound, "documentation page not found", nil)
			return
		}
		respond(c, http.StatusInternalServerError, "failed to read documentation content", nil)
		return
	}

	respond(c, http.StatusOK, "Documentation content fetched successfully", gin.H{
		"slug":    slug,
		"content": content,
	})
}
