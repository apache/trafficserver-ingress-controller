package watcher

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/apache/trafficserver-ingress-controller/endpoint"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AtsCacheHandler handles ATSCachingPolicy events
type AtsCacheHandler struct {
	ResourceName string
	Ep           *endpoint.Endpoint
}

// Constructor
func NewAtsCacheHandler(resource string, ep *endpoint.Endpoint) *AtsCacheHandler {
	log.Println("ATS Cache Constructor initialized ")
	return &AtsCacheHandler{ResourceName: resource, Ep: ep}
}

func (h *AtsCacheHandler) filePath() string {
	return "/opt/ats/etc/trafficserver/cache.config"
}

// Update ATS config
func (h *AtsCacheHandler) UpdateAts() {
	log.Println("Update ATS called")
	msg, err := h.Ep.ATSManager.CacheSet()
	if err != nil {
		log.Println("UpdateAts error:", err)
	} else {
		log.Println("ATS updated:", msg)
	}
}

// Add handles creation of ATSCachingPolicy resources
func (h *AtsCacheHandler) Add(obj interface{}) {
	u := obj.(*unstructured.Unstructured)
	log.Printf("[ADD] ATSCachingPolicy: %s/%s", u.GetNamespace(), u.GetName())

	rules, found, err := unstructured.NestedSlice(u.Object, "spec", "rules")
	if err != nil || !found {
		log.Printf("Add: rules not found or error occurred: %v", err)
		return
	}

	var lines []string
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		primary, found, _ := unstructured.NestedMap(ruleMap, "primarySpecifier")
		if !found {
			continue
		}

		typeval, ok1 := primary["type"].(string)
		pattern, ok2 := primary["pattern"].(string)
		action, ok3 := ruleMap["action"].(string)
		ttl, ok4 := ruleMap["ttl"].(string)

		if !ok1 || !ok2 || !ok3 || !ok4 {
			continue
		}

		if action == "cache" {
			line := fmt.Sprintf("%s=%s ttl-in-cache=%s", typeval, pattern, ttl)
			lines = append(lines, line)
		}
	}

	configPath := h.filePath()
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Add: Failed to open cache.config: %v", err)
		return
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			log.Printf("Add: Failed to write line to cache.config: %v", err)
		}
	}

	h.UpdateAts()
}

// Update handles updates to ATSCachingPolicy resources
func (h *AtsCacheHandler) Update(oldObj, newObj interface{}) {
	newU := newObj.(*unstructured.Unstructured)
	log.Printf("[UPDATE] ATSCachingPolicy: %s/%s", newU.GetNamespace(), newU.GetName())

	newRules, found, err := unstructured.NestedSlice(newU.Object, "spec", "rules")
	if err != nil || !found {
		log.Printf("Update: rules not found or error occurred: %v", err)
		return
	}

	configPath := h.filePath()
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Update: Failed to read cache.config: %v", err)
		return
	}
	lines := strings.Split(string(existingData), "\n")

	for _, rule := range newRules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}
		primary, found, _ := unstructured.NestedMap(ruleMap, "primarySpecifier")
		if !found {
			continue
		}

		typeval, ok1 := primary["type"].(string)
		pattern, ok2 := primary["pattern"].(string)
		action, ok3 := ruleMap["action"].(string)
		newTTL, ok4 := ruleMap["ttl"].(string)

		if !ok1 || !ok2 || !ok3 || !ok4 || action != "cache" {
			continue
		}

		for i, line := range lines {
			if strings.Contains(line, fmt.Sprintf("%s=%s", typeval, pattern)) {
				lines[i] = fmt.Sprintf("%s=%s ttl-in-cache=%s", typeval, pattern, newTTL)
				break
			}
		}
	}

	err = os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		log.Printf("Update: Failed to write updated cache.config: %v", err)
	}
	h.UpdateAts()
}

// Delete handles deletion of ATSCachingPolicy resources
func (h *AtsCacheHandler) Delete(obj interface{}) {
	u := obj.(*unstructured.Unstructured)
	log.Printf("[DELETE] ATSCachingPolicy: %s/%s", u.GetNamespace(), u.GetName())

	configPath := h.filePath()
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Delete: Failed to read cache.config: %v", err)
		return
	}
	lines := strings.Split(string(existingData), "\n")

	rules, found, err := unstructured.NestedSlice(u.Object, "spec", "rules")
	if err != nil || !found {
		log.Printf("Delete: rules not found or error occurred: %v", err)
		return
	}

	patternsToDelete := make(map[string]string)
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}
		primary, found, _ := unstructured.NestedMap(ruleMap, "primarySpecifier")
		if !found {
			continue
		}

		typeval, ok1 := primary["type"].(string)
		pattern, ok2 := primary["pattern"].(string)
		action, ok3 := ruleMap["action"].(string)

		if ok1 && ok2 && ok3 && action == "cache" {
			patternsToDelete[typeval] = pattern
		}
	}

	var updatedLines []string
	for _, line := range lines {
		shouldDelete := false
		for typeval, pattern := range patternsToDelete {
			if strings.Contains(line, fmt.Sprintf("%s=%s", typeval, pattern)) {
				shouldDelete = true
				break
			}
		}
		if !shouldDelete {
			updatedLines = append(updatedLines, line)
		}
	}

	err = os.WriteFile(configPath, []byte(strings.Join(updatedLines, "\n")), 0644)
	if err != nil {
		log.Printf("Delete: Failed to write updated cache.config: %v", err)
	}

	h.UpdateAts()
}
