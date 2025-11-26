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
			"kind":       "TrafficServerSNIConfig",
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

// verifyFqdnOrder ensures fqdn is the first key in each entry
func verifyFqdnOrder(t *testing.T, path string) {
	data, _ := os.ReadFile(path)
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("failed to parse yaml nodes: %v", err)
	}
	// Find the "sni" sequence node
	for i := 0; i < len(root.Content); i++ {
		if root.Content[i].Value == "sni" {
			sniNode := root.Content[i+1]
			for _, item := range sniNode.Content {
				if len(item.Content) == 0 {
					continue
				}
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
