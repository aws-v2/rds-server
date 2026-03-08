package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DocPage represents a single documentation page in the manifest
type DocPage struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// DocsManifest represents the collection of available documentation pages
type DocsManifest struct {
	Pages []DocPage `json:"pages"`
}

// DocsService handles retrieval of documentation manifest and content
type DocsService struct {
	docsDir string
}

// NewDocsService creates a new documentation service
func NewDocsService(docsDir string) *DocsService {
	return &DocsService{docsDir: docsDir}
}

// GetManifest returns the list of available documentation pages
func (s *DocsService) GetManifest(ctx context.Context) (*DocsManifest, error) {
	files, err := os.ReadDir(s.docsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read docs directory: %w", err)
	}

	manifest := &DocsManifest{
		Pages: []DocPage{},
	}

	// Define desired order of slugs
	order := []string{
		"rds-overview",
		"rds-create",
		"rds-modify",
		"rds-clusters",
		"rds-snapshots",
		"rds-volumes",
		"rds-restore",
	}

	// Map for quick lookup
	fileMap := make(map[string]bool)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			fileMap[strings.TrimSuffix(file.Name(), ".md")] = true
		}
	}

	// Add pages in defined order if file exists
	for _, slug := range order {
		if fileMap[slug] {
			manifest.Pages = append(manifest.Pages, DocPage{
				Slug:  slug,
				Title: s.formatTitle(slug),
			})
			delete(fileMap, slug)
		}
	}

	// Add any remaining files
	for slug := range fileMap {
		manifest.Pages = append(manifest.Pages, DocPage{
			Slug:  slug,
			Title: s.formatTitle(slug),
		})
	}

	return manifest, nil
}

// GetDocContent returns the markdown content for a specific slug
func (s *DocsService) GetDocContent(ctx context.Context, slug string) (string, error) {
	// Security: basic check to prevent directory traversal
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return "", fmt.Errorf("invalid slug format")
	}

	filePath := filepath.Join(s.docsDir, slug+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("documentation not found")
		}
		return "", fmt.Errorf("failed to read documentation file: %w", err)
	}

	return string(content), nil
}

func (s *DocsService) formatTitle(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	title := strings.Join(parts, " ")
	// Special handling for RDS prefix
	if strings.HasPrefix(strings.ToLower(title), "rds ") {
		title = "RDS " + title[4:]
	}
	return title
}
