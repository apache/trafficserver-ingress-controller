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
	//"encoding/json"
	"fmt"
	"log"

	"ingress-ats/endpoint"
	//	t "ingress-ats/types"
	"ingress-ats/util"

	v1 "k8s.io/api/core/v1"
	//	set "github.com/deckarep/golang-set"
)

// EpHandler implements EventHandler
type EpHandler struct {
	ResourceName string
	Ep           *endpoint.Endpoint
}

func (e *EpHandler) Add(obj interface{}) {
	log.Printf("\n\nIn EndpointHandler ADD %#v \n\n", obj)
	e.add(obj)
	e.Ep.RedisClient.PrintAllKeys()
}

func (e *EpHandler) add(obj interface{}) {
	eps, ok := obj.(*v1.Endpoints)
	if !ok {
		log.Println("In EndpointHandler Add; cannot cast to *v1.Endpoints.")
		return
	}
	podSvcName := eps.GetObjectMeta().GetName()
	namespace := eps.GetNamespace()

	if !e.Ep.NsManager.IncludeNamespace(namespace) {
		log.Println("Namespace not included")
		return
	}

	for _, subset := range eps.Subsets {
		for _, port := range subset.Ports {
			portnum := fmt.Sprint(port.Port)
			portname := port.Name
			key := util.ConstructSvcPortString(namespace, podSvcName, portnum)
			for _, addr := range subset.Addresses {
				v := util.ConstructIPPortString(addr.IP, portnum, portname)
				e.Ep.RedisClient.DefaultDBSAdd(key, v)
			}
		}

	}
}

// Update for EventHandler
func (e *EpHandler) Update(obj, newObj interface{}) {
	log.Printf("\n\nEndpoint Update\n Obj: %#v \n newObj: %#v", obj, newObj)
	e.update(newObj)
	e.Ep.RedisClient.PrintAllKeys()
}

func (e *EpHandler) update(obj interface{}) {
	eps, ok := obj.(*v1.Endpoints)
	if !ok {
		log.Println("In EndpointHandler Add; cannot cast to *v1.Endpoints.")
		return
	}
	podSvcName := eps.GetObjectMeta().GetName()
	namespace := eps.GetNamespace()

	if !e.Ep.NsManager.IncludeNamespace(namespace) {
		log.Println("Namespace not included")
		return
	}

	for _, subset := range eps.Subsets {
		for _, port := range subset.Ports {
			portnum := fmt.Sprint(port.Port)
			portname := port.Name
			key := util.ConstructSvcPortString(namespace, podSvcName, portnum)
			for _, addr := range subset.Addresses {
				k := "temp_" + key
				v := util.ConstructIPPortString(addr.IP, portnum, portname)
				e.Ep.RedisClient.DefaultDBSAdd(k, v)
			}
			e.Ep.RedisClient.DefaultDBSUnionStore(key, "temp_"+key)
			e.Ep.RedisClient.DefaultDBDel("temp_" + key)
		}

	}
}

// Delete for EventHandler
func (e *EpHandler) Delete(obj interface{}) {
	log.Printf("\n\nEndpoint Delete: %#v \n\n", obj)
	e.delete(obj)
	e.Ep.RedisClient.PrintAllKeys()
}

func (e *EpHandler) delete(obj interface{}) {
	eps, ok := obj.(*v1.Endpoints)
	if !ok {
		log.Println("In EndpointHandler DELETE; cannot cast to *v1.Endpoints.")
		return
	}
	podSvcName := eps.GetObjectMeta().GetName()
	namespace := eps.GetNamespace()

	if !e.Ep.NsManager.IncludeNamespace(namespace) {
		log.Println("Namespace not included")
		return
	}

	for _, subset := range eps.Subsets {
		for _, port := range subset.Ports {
			portnum := fmt.Sprint(port.Port)
			key := util.ConstructSvcPortString(namespace, podSvcName, portnum)
			e.Ep.RedisClient.DefaultDBDel(key)
		}

	}

}

// GetResourceName returns the resource name
func (e *EpHandler) GetResourceName() string {
	return e.ResourceName
}
