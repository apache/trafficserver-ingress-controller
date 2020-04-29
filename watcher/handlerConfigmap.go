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

	"ingress-ats/endpoint"
	//	t "ingress-ats/types"

	v1 "k8s.io/api/core/v1"
)

// CMHandler handles Add Update Delete methods on Configmaps
type CMHandler struct {
	ResourceName string
	Ep           *endpoint.Endpoint
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
	for currKey, currVal := range cm.Data {
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
	return
}

// GetResourceName returns the resource name
func (c *CMHandler) GetResourceName() string {
	return c.ResourceName
}
