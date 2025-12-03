package watcher

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/proxy"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestSniHandler creates a temporary AtsSniHandler for testing.
// It overrides FilePath to point to a temp sni.yaml file.
func newTestSniHandler(t *testing.T) (*AtsSniHandler, string) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sni.yaml")

	if err := os.WriteFile(tmpFile, []byte("sni:\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ep := createExampleEndpointWithFakeATSSni()
	h := NewAtsSniHandler("test-resource", &ep, tmpFile)
	return h, tmpFile
}

// newSniConfig creates a CRD-like unstructured object for test
func newSniConfig(name string, fqdns []string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "trafficserver.apache.org/v1alpha1",
			"kind":       "ATSSniPolicy",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"sni": []interface{}{},
			},
		},
	}
	var rules []interface{}
	for _, fqdn := range fqdns {
		rules = append(rules, map[string]interface{}{
			"fqdn":                  fqdn,
			"verify_client":         "STRICT",
			"host_sni_policy":       "PERMISSIVE",
			"valid_tls_versions_in": []interface{}{"TLSv1_2", "TLSv1_3"},
		})
	}
	_ = unstructured.SetNestedSlice(u.Object, rules, "spec", "sni")
	return u
}

// parseSniYaml parses the written YAML file into []map[string]interface{}
func parseSniYaml(t *testing.T, path string) []map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if len(data) == 0 {
		return nil
	}
	var parsed struct {
		Sni []map[string]interface{} `yaml:"sni"`
	}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v\nData:\n%s", err, string(data))
	}
	return parsed.Sni
}

// TestAddSni verifies h.Add() adds fqdn entries into sni.yaml
func TestAddSni(t *testing.T) {
	h, tmpFile := newTestSniHandler(t)
	obj := newSniConfig("my-sni-config", []string{"ats.test.com", "host-test.com"})

	h.Add(obj)

	entries := parseSniYaml(t, tmpFile)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, entry := range entries {
		if _, ok := entry["fqdn"]; !ok {
			t.Errorf("missing fqdn field in entry: %+v", entry)
		}
	}
	verifyFqdnOrder(t, tmpFile)
}

// TestUpdateSni verifies h.Update() updates fqdn rules and removes old ones
func TestUpdateSni(t *testing.T) {
	h, tmpFile := newTestSniHandler(t)

	oldObj := newSniConfig("my-sni-config", []string{"ats.test.com", "host-test.com"})
	h.Add(oldObj)

	newObj := newSniConfig("my-sni-config", []string{"ats.test.com", "new-host.com"})
	h.Update(oldObj, newObj)

	entries := parseSniYaml(t, tmpFile)
	found := map[string]bool{}
	for _, e := range entries {
		fqdn := e["fqdn"].(string)
		found[fqdn] = true
	}

	for _, fqdn := range []string{"ats.test.com", "new-host.com"} {
		if !found[fqdn] {
			t.Errorf("expected fqdn %q not found", fqdn)
		}
	}
	if found["old-host.com"] {
		t.Errorf("unexpected fqdn old-host.com found")
	}
	verifyFqdnOrder(t, tmpFile)
}

// TestDeleteSni verifies h.Delete() removes fqdn rules from sni.yaml
func TestDeleteSni(t *testing.T) {
	h, tmpFile := newTestSniHandler(t)

	addObj := newSniConfig("my-sni-config", []string{"ats.test.com", "keep-me.com"})
	h.Add(addObj)

	delObj := newSniConfig("my-sni-config", []string{"ats.test.com"})
	h.Delete(delObj)

	entries := parseSniYaml(t, tmpFile)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if fqdn := entries[0]["fqdn"]; fqdn != "keep-me.com" {
		t.Errorf("expected keep-me.com to remain, got %v", fqdn)
	}

	delObj2 := newSniConfig("my-sni-config", []string{"keep-me.com"})
	h.Delete(delObj2)

	data, _ := os.ReadFile(tmpFile)
	if len(data) != 0 {
		t.Errorf("expected empty sni.yaml, got:\n%s", string(data))
	}
}

// TestLoadWriteSniFile verifies roundtrip of writeSniFile and loadSniFile
func TestLoadWriteSniFile(t *testing.T) {
	h, tmpFile := newTestSniHandler(t)

	expected := SniFile{
		Sni: []SniEntry{
			{"fqdn": "abc.com", "verify_client": "STRICT"},
		},
	}
	h.writeSniFile(expected)

	got := h.loadSniFile()
	if !reflect.DeepEqual(expected.Sni, got.Sni) {
		t.Errorf("expected %+v, got %+v", expected.Sni, got.Sni)
	}

	h.writeSniFile(SniFile{})
	data, _ := os.ReadFile(tmpFile)
	if len(data) != 0 {
		t.Errorf("expected empty file, got: %s", string(data))
	}
}

// TestArrayPreservation verifies that arrays (e.g. valid_tls_versions_in) are preserved as native YAML sequences
func TestArrayPreservation(t *testing.T) {
	h, tmpFile := newTestSniHandler(t)

	obj := newSniConfig("array-test", []string{"arr.test.com"})
	h.Add(obj)

	entries := parseSniYaml(t, tmpFile)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	v, ok := entries[0]["valid_tls_versions_in"]
	if !ok {
		t.Fatalf("missing valid_tls_versions_in in entry: %+v", entries[0])
	}

	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("expected valid_tls_versions_in to be []interface{}, got %T", v)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements in valid_tls_versions_in, got %d", len(arr))
	}
	if arr[0].(string) != "TLSv1_2" || arr[1].(string) != "TLSv1_3" {
		t.Errorf("unexpected values in valid_tls_versions_in: %v", arr)
	}
}

// verifyFqdnOrder ensures fqdn is the first key in each entry
func verifyFqdnOrder(t *testing.T, path string) {
	data, _ := os.ReadFile(path)
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("failed to parse yaml nodes: %v", err)
	}

	// Descend to the top-level mapping node (document -> mapping)
	var mappingNode *yaml.Node
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		mappingNode = root.Content[0]
	} else {
		mappingNode = &root
	}

	// mappingNode.Content contains key/value node pairs
	for i := 0; i < len(mappingNode.Content); i += 2 {
		keyNode := mappingNode.Content[i]
		valNode := mappingNode.Content[i+1]
		if keyNode.Value == "sni" && valNode.Kind == yaml.SequenceNode {
			// iterate sequence items (each item should be a mapping node)
			for _, item := range valNode.Content {
				if item.Kind != yaml.MappingNode || len(item.Content) == 0 {
					continue
				}
				// mapping node Content is key/value pairs; first key is at index 0
				firstKey := item.Content[0].Value
				if firstKey != "fqdn" {
					t.Errorf("expected fqdn as first key, got %q in %v", firstKey, item)
				}
			}
		}
	}
}

// Fake ATS Endpoint
func createExampleEndpointWithFakeATSSni() endpoint.Endpoint {
	return endpoint.Endpoint{
		ATSManager: &proxy.FakeATSManager{
			Namespace:    "default",
			IngressClass: "",
			Config:       make(map[string]string),
		},
	}
}
