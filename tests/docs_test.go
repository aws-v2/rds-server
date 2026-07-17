package health_test

import (
 
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"rds/internal/application"
	"rds/internal/middleware"
	http_transport "rds/internal/transport/http"
)

func TestDocsEndpoints(t *testing.T) {
	// 1. Create a temp directory for test documentation assets
	tempDir := t.TempDir()

	publicDir := filepath.Join(tempDir, "public")
	internalDir := filepath.Join(tempDir, "internal")

	err := os.MkdirAll(publicDir, 0755)
	if err != nil {
		t.Fatalf("failed to create public dir: %v", err)
	}

	err = os.MkdirAll(internalDir, 0755)
	if err != nil {
		t.Fatalf("failed to create internal dir: %v", err)
	}

	// 2. Write public and internal manifest files
	pubManifest := `{"service":"test-service","version":"1","categories":[{"title":"PubCat","items":[{"title":"PubItem","slug":"pub-slug"}]}]}`
	intManifest := `{"service":"test-service","version":"1","categories":[{"title":"IntCat","items":[{"title":"IntItem","slug":"int-slug"}]}]}`

	err = os.WriteFile(filepath.Join(publicDir, "manifest.json"), []byte(pubManifest), 0644)
	if err != nil {
		t.Fatalf("failed to write public manifest: %v", err)
	}

	err = os.WriteFile(filepath.Join(internalDir, "manifest.json"), []byte(intManifest), 0644)
	if err != nil {
		t.Fatalf("failed to write internal manifest: %v", err)
	}

	// 3. Write public and internal document files
	pubDoc := "---\ntitle: Public Doc\ndescription: test desc\n---\nPublic Content"
	intDoc := "---\ntitle: Internal Doc\ndescription: test desc\n---\nInternal Content"

	err = os.WriteFile(filepath.Join(publicDir, "pub-slug.md"), []byte(pubDoc), 0644)
	if err != nil {
		t.Fatalf("failed to write public doc: %v", err)
	}

	err = os.WriteFile(filepath.Join(internalDir, "int-slug.md"), []byte(intDoc), 0644)
	if err != nil {
		t.Fatalf("failed to write internal doc: %v", err)
	}

	// 4. Setup Gin engine with middleware and routes
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1/rds")
	v1.Use(middleware.AuthContextMiddleware())

	docsService := application.NewDocsService(tempDir)
	docsHandler := http_transport.NewDocsHandler(docsService)

	docsGroup := v1.Group("/docs")
	{
		docsGroup.GET("", docsHandler.GetManifest)
		docsGroup.GET("/:slug", docsHandler.GetDoc)
	}
 
// 	// 5. Test Manifest for role "USER"
// 	t.Run("Manifest for USER role", func(t *testing.T) {
// 		w, resp := sendReq("USER", "/docs")
// 		if w.Code != http.StatusOK {
// 			t.Errorf("expected 200, got %d", w.Code)
// 		}

// 		data, ok := resp["data"].(map[string]interface{})
// 		if !ok {
// 			t.Fatalf("response data is not map: %v", resp)
// 		}

// 		// categories, ok := data["categories"].([]interface{})
		
// 		categories, ok := response["public"].(map[string]interface{})["categories"].([]interface{})
// if !ok {
// 			t.Fatalf("categories is not slice: %v", data)
// 		}
// 		if len(categories) != 1 {
// 			t.Errorf("expected 1 category, got %d", len(categories))
// 		}

// 		cat0 := categories[0].(map[string]interface{})
// 		if cat0["title"] != "PubCat" {
// 			t.Errorf("expected PubCat category, got %s", cat0["title"])
// 		}
// 	})

	// // 6. Test Manifest for role other than USER (e.g., ADMIN)
	// t.Run("Manifest for ADMIN role", func(t *testing.T) {
	// 	w, resp := sendReq("ADMIN", "/docs")
	// 	if w.Code != http.StatusOK {
	// 		t.Errorf("expected 200, got %d", w.Code)
	// 	}

	// 	data, ok := resp["data"].(map[string]interface{})
	// 	if !ok {
	// 		t.Fatalf("response data is not map: %v", resp)
	// 	}

	// 	categories, ok := data["categories"].([]interface{})
	// 	if !ok {
	// 		t.Fatalf("categories is not slice: %v", data)
	// 	}

	// 	if len(categories) != 2 {
	// 		t.Errorf("expected 2 categories (merged), got %d", len(categories))
	// 	}

	// 	cat0 := categories[0].(map[string]interface{})
	// 	cat1 := categories[1].(map[string]interface{})
	// 	if cat0["title"] != "PubCat" || cat1["title"] != "IntCat" {
	// 		t.Errorf("unexpected categories: %s, %s", cat0["title"], cat1["title"])
	// 	}
	// })

	// // 7. Test Doc Access for role "USER" (Public slug)
	// t.Run("Public doc access for USER", func(t *testing.T) {
	// 	w, resp := sendReq("USER", "/docs/pub-slug")
	// 	if w.Code != http.StatusOK {
	// 		t.Errorf("expected 200, got %d", w.Code)
	// 	}

	// 	data, ok := resp["data"].(map[string]interface{})
	// 	if !ok {
	// 		t.Fatalf("response data is not map: %v", resp)
	// 	}

	// 	if data["content"] != "Public Content" {
	// 		t.Errorf("expected 'Public Content', got %v", data["content"])
	// 	}
	// })

	// // 8. Test Doc Access for role "USER" (Internal slug) -> Should return 404
	// t.Run("Internal doc access for USER (Forbidden)", func(t *testing.T) {
	// 	w, _ := sendReq("USER", "/docs/int-slug")
	// 	if w.Code != http.StatusNotFound {
	// 		t.Errorf("expected 404, got %d", w.Code)
	// 	}
	// })

	// // 9. Test Doc Access for role "ADMIN" (Internal slug) -> Should return 200
	// t.Run("Internal doc access for ADMIN", func(t *testing.T) {
	// 	w, resp := sendReq("ADMIN", "/docs/int-slug")
	// 	if w.Code != http.StatusOK {
	// 		t.Errorf("expected 200, got %d", w.Code)
	// 	}

	// 	data, ok := resp["data"].(map[string]interface{})
	// 	if !ok {
	// 		t.Fatalf("response data is not map: %v", resp)
	// 	}

	// 	if data["content"] != "Internal Content" {
	// 		t.Errorf("expected 'Internal Content', got %v", data["content"])
	// 	}
	// })
}
