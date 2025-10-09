package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/proxy"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestHandler creates a temporary AtsCacheHandler for testing.
// It overrides the handler's filePath to point to a temp cache.config file
// instead of the real /opt/ats/etc/trafficserver/cache.config.
func newTestHandler(t *testing.T) (*AtsCacheHandler, string) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "cache.config")

	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()
	// Ensure directory exists
	//os.MkdirAll(filepath.Dir(tmpFile), 0755)

	ep := createExampleEndpointWithFakeATSCache()
	h := NewAtsCacheHandler("test-resource", &ep, tmpFile)

	return h, tmpFile
}

// newCachingPolicy creates an unstructured ATSCachingPolicy object
// with the given name and rules. The rules must be []interface{} type.
func newCachingPolicy(name string, rules []interface{}) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.com/v1alpha1",
			"kind":       "ATSCachingPolicy",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"rules": rules,
			},
		},
	}
	return u
}

// TestAddCachingPolicy verifies that calling h.Add(policy)
// writes the expected caching rule to cache.config and reloads configurations.
func TestAddCachingPolicy(t *testing.T) {
	h, tmpFile := newTestHandler(t)

	rules := []interface{}{
		map[string]interface{}{
			"primarySpecifier": map[string]interface{}{
				"type":    "url_regex",
				"pattern": "/images/.*",
			},
			"action": "cache",
			"ttl":    "3600s",
		},
	}
	policy := newCachingPolicy("policy1", rules)

	h.Add(policy)

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read cache.config: %v", err)
	}
	content := string(data)
	if content == "" || !containsLine(content, "url_regex=/images/.* ttl-in-cache=3600s") {
		t.Errorf("expected cache.config to contain rule, got: %s", content)
	}
}

// TestUpdateCachingPolicy verifies that calling h.Update(nil, newPolicy)
// modifies the existing caching rule in cache.config with new values and reloads configurations.
func TestUpdateCachingPolicy(t *testing.T) {
	h, tmpFile := newTestHandler(t)

	// Initial rule
	initial := "url_regex=/images/.* ttl-in-cache=3600s\n"
	if err := os.WriteFile(tmpFile, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to setup initial cache.config: %v", err)
	}

	// Update rule with new TTL
	rules := []interface{}{
		map[string]interface{}{
			"primarySpecifier": map[string]interface{}{
				"type":    "url_regex",
				"pattern": "/images/.*",
			},
			"action": "cache",
			"ttl":    "7200s",
		},
	}
	newPolicy := newCachingPolicy("policy1", rules)

	h.Update(nil, newPolicy)

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read cache.config: %v", err)
	}
	content := string(data)
	if !containsLine(content, "url_regex=/images/.* ttl-in-cache=7200s") {
		t.Errorf("expected updated TTL, got: %s", content)
	}
}

// TestDeleteCachingPolicy verifies that calling h.Delete(policy)
// removes the matching caching rule from cache.config, but keeps unrelated lines intact and reloads configurations.
func TestDeleteCachingPolicy(t *testing.T) {
	h, tmpFile := newTestHandler(t)

	initial := "url_regex=/images/.* ttl-in-cache=3600s\nother_line=keepme\n"
	if err := os.WriteFile(tmpFile, []byte(initial), 0644); err != nil {
		t.Fatalf("failed to setup initial cache.config: %v", err)
	}

	rules := []interface{}{
		map[string]interface{}{
			"primarySpecifier": map[string]interface{}{
				"type":    "url_regex",
				"pattern": "/images/.*",
			},
			"action": "cache",
			"ttl":    "3600s",
		},
	}
	policy := newCachingPolicy("policy1", rules)

	h.Delete(policy)

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read cache.config: %v", err)
	}
	content := string(data)
	if containsLine(content, "url_regex=/images/.* ttl-in-cache=3600s") {
		t.Errorf("expected rule to be deleted, got: %s", content)
	}
	if !containsLine(content, "other_line=keepme") {
		t.Errorf("expected unrelated lines to remain, got: %s", content)
	}
}

// containsLine checks if the given line exists in content.
func containsLine(content, line string) bool {
	for _, l := range splitLines(content) {
		if l == line {
			return true
		}
	}
	return false
}

// splitLines splits a string by newline into individual lines.
func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// createExampleEndpointWithFakeATSCache creates a fake Endpoint with a FakeATSManager,
// used for unit testing without a real Traffic Server or Redis.
func createExampleEndpointWithFakeATSCache() endpoint.Endpoint {
	ep := endpoint.Endpoint{
		ATSManager: &proxy.FakeATSManager{
			Namespace:    "default",
			IngressClass: "",
			Config:       make(map[string]string),
		},
	}
	return ep
}
