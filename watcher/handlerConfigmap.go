/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package watcher

import (
	"log"
	"os"
	"strings"

	"github.com/apache/trafficserver-ingress-controller/endpoint"

	v1 "k8s.io/api/core/v1"
)

// CMHandler handles Add Update Delete methods on Configmaps
type CMHandler struct {
	ResourceName string
	Ep           *endpoint.Endpoint
}

// configMapAllowlistEnv is the environment variable through which the
// operator may extend the built-in allowlist of records.config keys that an
// annotated ConfigMap is permitted to set. It holds a comma-separated list
// of entries; an entry ending in "*" matches any key with that prefix, any
// other entry must match a key exactly. Entries must begin with
// "proxy.config." - anything else is ignored and logged. The variable lives
// in the pod spec, so extending the allowlist requires write access to the
// Deployment: a strictly higher privilege than the ConfigMap write access
// this allowlist defends against.
const configMapAllowlistEnv = "CONFIGMAP_RECORD_ALLOWLIST"

// builtinAllowedRecords is the fail-closed default allowlist of
// records.config keys that a ConfigMap may set on the live proxy. The
// ats-configmap annotation is attacker-suppliable by anyone who can write
// ConfigMaps in a watched namespace, so only operationally safe tuning
// knobs are allowed by default. Records such as proxy.config.ssl.*,
// proxy.config.http.push_method_enabled, proxy.config.log.*,
// proxy.config.diags.* and proxy.config.plugin.* must never be settable
// through this path unless the operator explicitly opts them in via
// CONFIGMAP_RECORD_ALLOWLIST on the pod spec.
var builtinAllowedRecords = []string{
	"proxy.config.output.logfile.rolling_*",
	"proxy.config.restart.*",
	"proxy.config.http.keep_alive_*",
	"proxy.config.http.transaction_no_activity_timeout_*",
	"proxy.config.http.transaction_active_timeout_*",
	"proxy.config.http.connect_attempts_*",
	"proxy.config.http.parent_proxy.connect_attempts_timeout",
	"proxy.config.net.connections_throttle",
	"proxy.config.cache.ram_cache.size",
}

// matchesAllowlistEntry reports whether key matches one allowlist entry:
// prefix match when the entry ends in "*", exact match otherwise.
func matchesAllowlistEntry(key, entry string) bool {
	if strings.HasSuffix(entry, "*") {
		return strings.HasPrefix(key, strings.TrimSuffix(entry, "*"))
	}
	return key == entry
}

// configKeyAllowed reports whether a records.config key coming from a
// ConfigMap is on the built-in allowlist or in the operator-supplied
// extension carried by CONFIGMAP_RECORD_ALLOWLIST.
func configKeyAllowed(key string) bool {
	for _, entry := range builtinAllowedRecords {
		if matchesAllowlistEntry(key, entry) {
			return true
		}
	}
	for _, entry := range strings.Split(os.Getenv(configMapAllowlistEnv), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.HasPrefix(entry, "proxy.config.") {
			log.Printf("Ignoring %s entry %q: allowlist entries must start with proxy.config.", configMapAllowlistEnv, entry)
			continue
		}
		if matchesAllowlistEntry(key, entry) {
			return true
		}
	}
	return false
}

// configValueAllowed rejects values that do not look like a single
// records.config token. traffic_ctl is exec'd argv-style so there is no
// shell to inject into, but control characters and whitespace have no
// place in a config value and only help smuggle surprises downstream.
func configValueAllowed(value string) bool {
	return !strings.ContainsAny(value, " \t\r\n\x00")
}

// Add for EventHandler
func (c *CMHandler) Add(obj interface{}) {
	c.update(obj)
}

func (c *CMHandler) update(newObj interface{}) {
	cm, ok := newObj.(*v1.ConfigMap)
	if !ok {
		log.Println("In ConfigMapHandler Update; cannot cast to *v1.ConfigMap")
		return
	}

	annotations := cm.GetAnnotations()
	if val, ok := annotations["ats-configmap"]; ok {
		if val != "true" {
			return
		}
	} else {
		return
	}

	for currKey, currVal := range cm.Data {
		if !configKeyAllowed(currKey) {
			log.Printf("Rejected config key %q from ConfigMap %s/%s: not in the allowlist of safe reloadable records (extend via %s in the pod spec if this key is intended)", currKey, cm.GetNamespace(), cm.GetName(), configMapAllowlistEnv)
			continue
		}
		if !configValueAllowed(currVal) {
			log.Printf("Rejected config value for key %q from ConfigMap %s/%s: contains whitespace or control characters", currKey, cm.GetNamespace(), cm.GetName())
			continue
		}
		msg, err := c.Ep.ATSManager.ConfigSet(currKey, currVal) // update ATS
		if err != nil {
			log.Println(err)
		} else {
			log.Println(msg)
		}
	}
}

// Update for EventHandler
func (c *CMHandler) Update(obj, newObj interface{}) {
	c.update(newObj)
}

// Delete for EventHandler
func (c *CMHandler) Delete(obj interface{}) {
	// do not handle delete events for now
}

// GetResourceName returns the resource name
func (c *CMHandler) GetResourceName() string {
	return c.ResourceName
}
