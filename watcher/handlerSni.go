package watcher

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"

	"github.com/apache/trafficserver-ingress-controller/endpoint"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AtsSniHandler handles TrafficServerSNIConfig CR events
type AtsSniHandler struct {
	ResourceName string
	Ep           *endpoint.Endpoint
	FilePath     string
}

// Constructor
func NewAtsSniHandler(resource string, ep *endpoint.Endpoint, path string) *AtsSniHandler {
	log.Println("Ats SNI Handler initialized")
	return &AtsSniHandler{ResourceName: resource, Ep: ep, FilePath: path}
}

// SniEntry represents one fqdn entry in sni.yaml (flexible, dynamic)
type SniEntry map[string]interface{}

// Custom YAML marshaller to ensure fqdn appears first
func (s SniEntry) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	// Write fqdn first if present
	if fqdn, ok := s["fqdn"]; ok {
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "fqdn"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", fqdn)},
		)
	}

	// Collect remaining keys (excluding fqdn)
	keys := make([]string, 0, len(s))
	for k := range s {
		if k != "fqdn" {
			keys = append(keys, k)
		}
	}

	// Sort the other keys alphabetically for consistent output
	sort.Strings(keys)

	// Append other keys
	for _, k := range keys {
		v := s[k]
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", v)},
		)
	}

	return node, nil
}

// SniFile represents the full sni.yaml structure
type SniFile struct {
	Sni []SniEntry `yaml:"sni,omitempty"`
}

// Add handles creation of TrafficServerSNIConfig
func (h *AtsSniHandler) Add(obj interface{}) {
	u := obj.(*unstructured.Unstructured)
	log.Printf("[ADD] TrafficServerSNIConfig: %s/%s", u.GetNamespace(), u.GetName())

	newSni, found, err := unstructured.NestedSlice(u.Object, "spec", "sni")
	if err != nil || !found {
		log.Printf("Add: sni not found or error: %v", err)
		return
	}

	sniFile := h.loadSniFile()

	for _, entry := range newSni {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		fqdn, ok := entryMap["fqdn"].(string)
		if !ok || fqdn == "" {
			continue
		}

		updated := false
		for i, existing := range sniFile.Sni {
			if existingFqdn, _ := existing["fqdn"].(string); existingFqdn == fqdn {
				if !reflect.DeepEqual(existing, entryMap) {
					sniFile.Sni[i] = entryMap
				}
				updated = true
				break
			}
		}
		if !updated {
			sniFile.Sni = append(sniFile.Sni, entryMap)
		}
	}

	h.writeSniFile(sniFile)
	h.reloadSni()
}

// Update handles updates of TrafficServerSNIConfig
func (h *AtsSniHandler) Update(oldObj, newObj interface{}) {
	newU := newObj.(*unstructured.Unstructured)
	log.Printf("[UPDATE] TrafficServerSNIConfig: %s/%s", newU.GetNamespace(), newU.GetName())

	newSni, found, err := unstructured.NestedSlice(newU.Object, "spec", "sni")
	if err != nil || !found {
		log.Printf("Update: sni not found or error: %v", err)
		return
	}

	sniFile := h.loadSniFile()

	newMap := make(map[string]SniEntry)
	for _, entry := range newSni {
		if m, ok := entry.(map[string]interface{}); ok {
			if fqdn, ok := m["fqdn"].(string); ok && fqdn != "" {
				newMap[fqdn] = m
			}
		}
	}

	log.Println("New Updated map in Update function ", newMap)
	var updatedSni []SniEntry
	seen := make(map[string]struct{})

	for _, existing := range sniFile.Sni {
		fqdn, _ := existing["fqdn"].(string)
		if newEntry, ok := newMap[fqdn]; ok {
			updatedSni = append(updatedSni, newEntry)
			seen[fqdn] = struct{}{}
		} else {
			updatedSni = append(updatedSni, existing)
		}
	}

	for fqdn, newEntry := range newMap {
		if _, already := seen[fqdn]; !already {
			updatedSni = append(updatedSni, newEntry)
		}
	}

	sniFile.Sni = updatedSni
	h.writeSniFile(sniFile)
	h.reloadSni()
}

// Delete handles deletion of TrafficServerSNIConfig
func (h *AtsSniHandler) Delete(obj interface{}) {
	u := obj.(*unstructured.Unstructured)
	log.Printf("[DELETE] TrafficServerSNIConfig: %s/%s", u.GetNamespace(), u.GetName())

	sniFile := h.loadSniFile()

	sniList, found, err := unstructured.NestedSlice(u.Object, "spec", "sni")
	if err != nil || !found {
		log.Printf("Delete: sni not found or error: %v", err)
		return
	}

	delMap := make(map[string]struct{})
	for _, entry := range sniList {
		if m, ok := entry.(map[string]interface{}); ok {
			if fqdn, ok := m["fqdn"].(string); ok && fqdn != "" {
				delMap[fqdn] = struct{}{}
			}
		}
	}

	var updatedSni []SniEntry
	for _, existing := range sniFile.Sni {
		fqdn, _ := existing["fqdn"].(string)
		if _, toDelete := delMap[fqdn]; !toDelete {
			updatedSni = append(updatedSni, existing)
		}
	}

	sniFile.Sni = updatedSni
	h.writeSniFile(sniFile)
	h.reloadSni()
}

// loadSniFile reads existing sni.yaml
func (h *AtsSniHandler) loadSniFile() SniFile {
	var sniFile SniFile
	data, err := os.ReadFile(h.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return sniFile
		}
		log.Printf("Failed to read sni.yaml: %v", err)
		return sniFile
	}
	if err := yaml.Unmarshal(data, &sniFile); err != nil {
		log.Printf("Failed to unmarshal sni.yaml: %v", err)
	}
	return sniFile
}

// writeSniFile writes sni.yaml atomically
func (h *AtsSniHandler) writeSniFile(sniFile SniFile) {
	if len(sniFile.Sni) == 0 {
		if err := os.WriteFile(h.FilePath, []byte{}, 0644); err != nil {
			log.Printf("Failed to clear sni.yaml: %v", err)
		}
		return
	}
	data, err := yaml.Marshal(&sniFile)
	if err != nil {
		log.Printf("Failed to marshal sni.yaml: %v", err)
		return
	}
	tmp := h.FilePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Printf("Failed to write temp sni.yaml: %v", err)
		return
	}
	_ = os.Rename(tmp, h.FilePath)
}

// reloadSni triggers ATS reload
func (h *AtsSniHandler) reloadSni() {
	if h.Ep != nil && h.Ep.ATSManager != nil {
		if msg, err := h.Ep.ATSManager.SniSet(); err != nil {
			log.Printf("Failed to reload ATS SNI: %v", err)
		} else {
			log.Printf("ATS SNI reloaded: %s", msg)
		}
	}
}
