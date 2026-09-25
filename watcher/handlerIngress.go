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
	"strconv"
	"sync"

	"github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/util"

	nv1 "k8s.io/api/networking/v1"
)

// IgHandler implements EventHandler
type IgHandler struct {
	ResourceName string
	Ep           *endpoint.Endpoint

	// hostOwners maps a hostname to the namespace that first claimed it and
	// a count of live Ingress claims from that namespace. A hostname's routes
	// may only be written by Ingresses of the owning namespace; conflicting
	// claims from other namespaces are rejected (first writer wins), so a
	// tenant cannot attach its backend to another tenant's hostname.
	// Held by pointer so copies of the handler share one registry and the
	// struct stays safe to copy.
	hostOwners *hostOwnerRegistry
}

type hostOwnerRegistry struct {
	mu     sync.Mutex
	owners map[string]*hostClaim
}

type hostClaim struct {
	namespace string
	refs      int
}

func (g *IgHandler) hostOwnerRegistryInit() *hostOwnerRegistry {
	if g.hostOwners == nil {
		g.hostOwners = &hostOwnerRegistry{owners: make(map[string]*hostClaim)}
	}
	return g.hostOwners
}

// claimHost records that an Ingress in namespace binds routes for host.
// It returns false when the host is already owned by a different namespace.
func (g *IgHandler) claimHost(host, namespace string) bool {
	r := g.hostOwnerRegistryInit()
	r.mu.Lock()
	defer r.mu.Unlock()

	claim, ok := r.owners[host]
	if !ok {
		r.owners[host] = &hostClaim{namespace: namespace, refs: 1}
		return true
	}
	if claim.namespace != namespace {
		return false
	}
	claim.refs++
	return true
}

// releaseHost drops one claim on host held by namespace. Releases by a
// namespace that does not own the host are ignored, so a rejected claimant
// cannot free another tenant's hostname by deleting its own Ingress.
func (g *IgHandler) releaseHost(host, namespace string) {
	r := g.hostOwnerRegistryInit()
	r.mu.Lock()
	defer r.mu.Unlock()

	claim, ok := r.owners[host]
	if !ok || claim.namespace != namespace {
		return
	}
	claim.refs--
	if claim.refs <= 0 {
		delete(r.owners, host)
	}
}

// ingressHosts returns the unique set of hostnames an Ingress binds routes
// for: "*" for the default backend and for rules with an empty host, plus
// every rule host.
func ingressHosts(ing *nv1.Ingress) []string {
	seen := make(map[string]bool)
	var hosts []string

	appendHost := func(host string) {
		if host == "" {
			host = "*"
		}
		if !seen[host] {
			seen[host] = true
			hosts = append(hosts, host)
		}
	}

	if ing.Spec.DefaultBackend != nil {
		appendHost("*")
	}
	for _, rule := range ing.Spec.Rules {
		appendHost(rule.Host)
	}
	return hosts
}

// claimIngressHosts claims every host the Ingress references and returns,
// per host, whether the claim succeeded. Rejected claims are logged.
func (g *IgHandler) claimIngressHosts(ing *nv1.Ingress, namespace, name string) map[string]bool {
	claimed := make(map[string]bool)
	for _, host := range ingressHosts(ing) {
		claimed[host] = g.claimHost(host, namespace)
		if !claimed[host] {
			log.Printf("Host %q in Ingress %s/%s is already owned by another namespace; skipping its routes", host, namespace, name)
		}
	}
	return claimed
}

func (g *IgHandler) Add(obj interface{}) {
	log.Printf("In INGRESS_HANDLER ADD %#v \n", obj)
	g.add(obj)
	g.Ep.RedisClient.PrintAllKeys()
}

func (g *IgHandler) add(obj interface{}) {
	ingressObj, ok := obj.(*nv1.Ingress)
	if !ok {
		log.Println("In HandlerIngress Add; cannot cast to *nv1.Ingress")
		return
	}

	namespace := ingressObj.GetNamespace()
	// v1.18 ingress class name field in ingress object
	//ingressClass, _ := util.ExtractIngressClass(ingressObj.GetAnnotations())
	ingressClass, _ := util.ExtractIngressClassName(obj)
	if !g.Ep.NsManager.IncludeNamespace(namespace) || !g.Ep.ATSManager.IncludeIngressClass(ingressClass) {
		log.Println("Namespace not included or Ingress Class not matched")
		return
	}

	name := ingressObj.GetName()
	version := ingressObj.GetResourceVersion()
	nameversion := util.ConstructNameVersionString(namespace, name, version)

	// add the script before adding route
	snippet, snippetErr := util.ExtractServerSnippet(ingressObj.GetAnnotations())
	if snippetErr == nil {
		log.Println("Snippet in the handlerIngress.go file: ", snippet)
		g.Ep.RedisClient.DBOneSAdd(nameversion, snippet)
	}

	// claim host ownership before writing any routes; hosts owned by a
	// different namespace are skipped
	claimed := g.claimIngressHosts(ingressObj, namespace, name)

	// add default backend rules
	if ingressObj.Spec.DefaultBackend != nil && claimed["*"] {
		host := "*"
		scheme := "http"
		path := "/"
		pathType := nv1.PathTypePrefix
		hostPath := util.ConstructHostPathString(scheme, host, path, pathType)
		service := ingressObj.Spec.DefaultBackend.Service.Name
		port := strconv.Itoa(int(ingressObj.Spec.DefaultBackend.Service.Port.Number))
		svcport := util.ConstructSvcPortString(namespace, service, port)

		g.Ep.RedisClient.DBOneSAdd(hostPath, svcport)

		if snippetErr == nil {
			g.Ep.RedisClient.DBOneSAdd(hostPath, nameversion)
		}

		// add default backend rule for https as well
		scheme = "https"
		hostPath = util.ConstructHostPathString(scheme, host, path, pathType)

		g.Ep.RedisClient.DBOneSAdd(hostPath, svcport)

		if snippetErr == nil {
			g.Ep.RedisClient.DBOneSAdd(hostPath, nameversion)
		}
	}

	tlsHosts := make(map[string]string)

	for _, ingressTLS := range ingressObj.Spec.TLS {
		for _, tlsHost := range ingressTLS.Hosts {
			tlsHosts[tlsHost] = "1"
		}
	}

	for _, ingressRule := range ingressObj.Spec.Rules {
		host := ingressRule.Host
		if host == "" {
			host = "*"
		}
		if !claimed[host] {
			continue
		}
		scheme := "http"
		if _, ok := tlsHosts[host]; ok {
			scheme = "https"
		}

		for _, httpPath := range ingressRule.HTTP.Paths {

			path := httpPath.Path
			pathType := *httpPath.PathType
			hostPath := util.ConstructHostPathString(scheme, host, path, pathType)
			service := httpPath.Backend.Service.Name
			port := strconv.Itoa(int(httpPath.Backend.Service.Port.Number))
			svcport := util.ConstructSvcPortString(namespace, service, port)

			g.Ep.RedisClient.DBOneSAdd(hostPath, svcport)

			if snippetErr == nil {
				g.Ep.RedisClient.DBOneSAdd(hostPath, nameversion)
			}
		}

	}
}

// Update for EventHandler
func (g *IgHandler) Update(obj, newObj interface{}) {
	log.Printf("In INGRESS_HANDLER UPDATE %#v \n", newObj)
	g.update(obj, newObj)
	g.Ep.RedisClient.PrintAllKeys()
}

func (g *IgHandler) update(obj, newObj interface{}) {
	ingressObj, ok := obj.(*nv1.Ingress)
	if !ok {
		log.Println("In HandlerIngress Update; cannot cast to *nv1.Ingress")
		return
	}

	newIngressObj, ok := newObj.(*nv1.Ingress)
	if !ok {
		log.Println("In HandlerIngress Update; cannot cast to *nv1.Ingress")
		return
	}

	m := make(map[string]string)

	namespace := ingressObj.GetNamespace()
	// v1.18 ingress class name field in ingress object
	//ingressClass, _ := util.ExtractIngressClass(ingressObj.GetAnnotations())
	ingressClass, _ := util.ExtractIngressClassName(obj)
	if g.Ep.NsManager.IncludeNamespace(namespace) && g.Ep.ATSManager.IncludeIngressClass(ingressClass) {
		log.Println("Old Namespace included")

		name := ingressObj.GetName()
		version := ingressObj.GetResourceVersion()
		nameversion := util.ConstructNameVersionString(namespace, name, version)

		_, snippetErr := util.ExtractServerSnippet(ingressObj.GetAnnotations())

		// handle default backend rules
		if ingressObj.Spec.DefaultBackend != nil {
			host := "*"
			scheme := "http"
			path := "/"
			pathType := nv1.PathTypePrefix
			hostPath := util.ConstructHostPathString(scheme, host, path, pathType)

			g.Ep.RedisClient.DBOneSUnionStore("temp_"+hostPath, hostPath)
			m["temp_"+hostPath] = hostPath

			service := ingressObj.Spec.DefaultBackend.Service.Name
			port := strconv.Itoa(int(ingressObj.Spec.DefaultBackend.Service.Port.Number))
			svcport := util.ConstructSvcPortString(namespace, service, port)

			g.Ep.RedisClient.DBOneSRem("temp_"+hostPath, svcport)

			if snippetErr == nil {
				g.Ep.RedisClient.DBOneSRem("temp_"+hostPath, nameversion)
			}

			// handle default backend https rule
			scheme = "https"
			hostPath = util.ConstructHostPathString(scheme, host, path, pathType)

			g.Ep.RedisClient.DBOneSUnionStore("temp_"+hostPath, hostPath)
			m["temp_"+hostPath] = hostPath

			g.Ep.RedisClient.DBOneSRem("temp_"+hostPath, svcport)

			if snippetErr == nil {
				g.Ep.RedisClient.DBOneSRem("temp_"+hostPath, nameversion)
			}
		}

		tlsHosts := make(map[string]string)

		for _, ingressTLS := range ingressObj.Spec.TLS {
			for _, tlsHost := range ingressTLS.Hosts {
				tlsHosts[tlsHost] = "1"
			}
		}

		for _, ingressRule := range ingressObj.Spec.Rules {
			host := ingressRule.Host
			if host == "" {
				host = "*"
			}
			scheme := "http"
			if _, ok := tlsHosts[host]; ok {
				scheme = "https"
			}

			for _, httpPath := range ingressRule.HTTP.Paths {

				path := httpPath.Path
				pathType := *httpPath.PathType
				hostPath := util.ConstructHostPathString(scheme, host, path, pathType)

				g.Ep.RedisClient.DBOneSUnionStore("temp_"+hostPath, hostPath)
				m["temp_"+hostPath] = hostPath

				service := httpPath.Backend.Service.Name
				port := strconv.Itoa(int(httpPath.Backend.Service.Port.Number))
				svcport := util.ConstructSvcPortString(namespace, service, port)

				g.Ep.RedisClient.DBOneSRem("temp_"+hostPath, svcport)

				if snippetErr == nil {
					g.Ep.RedisClient.DBOneSRem("temp_"+hostPath, nameversion)
				}
			}

		}

		// release the old spec's host claims; the new spec re-claims below
		for _, host := range ingressHosts(ingressObj) {
			g.releaseHost(host, namespace)
		}
	}

	newNamespace := newIngressObj.GetNamespace()
	// v1.18 ingress class name field in ingress object
	//newIngressClass, _ := util.ExtractIngressClass(newIngressObj.GetAnnotations())
	newIngressClass, _ := util.ExtractIngressClassName(newObj)
	if g.Ep.NsManager.IncludeNamespace(newNamespace) && g.Ep.ATSManager.IncludeIngressClass(newIngressClass) {
		log.Println("New Namespace included")

		name := newIngressObj.GetName()
		version := newIngressObj.GetResourceVersion()
		nameversion := util.ConstructNameVersionString(newNamespace, name, version)

		newSnippet, newSnippetErr := util.ExtractServerSnippet(newIngressObj.GetAnnotations())
		if newSnippetErr == nil {
			g.Ep.RedisClient.DBOneSAdd(nameversion, newSnippet)
		}

		// claim host ownership before writing any routes; hosts owned by a
		// different namespace are skipped
		claimed := g.claimIngressHosts(newIngressObj, newNamespace, name)

		// handle default backend rule
		if newIngressObj.Spec.DefaultBackend != nil && claimed["*"] {
			host := "*"
			scheme := "http"
			path := "/"
			pathType := nv1.PathTypePrefix
			hostPath := util.ConstructHostPathString(scheme, host, path, pathType)

			service := newIngressObj.Spec.DefaultBackend.Service.Name
			port := strconv.Itoa(int(newIngressObj.Spec.DefaultBackend.Service.Port.Number))
			svcport := util.ConstructSvcPortString(newNamespace, service, port)

			g.Ep.RedisClient.DBOneSAdd("temp_"+hostPath, svcport)
			m["temp_"+hostPath] = hostPath

			if newSnippetErr == nil {
				g.Ep.RedisClient.DBOneSAdd("temp_"+hostPath, nameversion)
			}

			// handle default backend rule for https as well
			scheme = "https"
			hostPath = util.ConstructHostPathString(scheme, host, path, pathType)

			g.Ep.RedisClient.DBOneSAdd("temp_"+hostPath, svcport)
			m["temp_"+hostPath] = hostPath

			if newSnippetErr == nil {
				g.Ep.RedisClient.DBOneSAdd("temp_"+hostPath, nameversion)
			}
		}

		newTlsHosts := make(map[string]string)

		for _, newIngressTLS := range newIngressObj.Spec.TLS {
			for _, newTlsHost := range newIngressTLS.Hosts {
				newTlsHosts[newTlsHost] = "1"
			}
		}

		for _, ingressRule := range newIngressObj.Spec.Rules {
			host := ingressRule.Host
			if host == "" {
				host = "*"
			}
			if !claimed[host] {
				continue
			}
			scheme := "http"
			if _, ok := newTlsHosts[host]; ok {
				scheme = "https"
			}

			for _, httpPath := range ingressRule.HTTP.Paths {

				path := httpPath.Path
				pathType := *httpPath.PathType
				hostPath := util.ConstructHostPathString(scheme, host, path, pathType)

				service := httpPath.Backend.Service.Name
				port := strconv.Itoa(int(httpPath.Backend.Service.Port.Number))
				svcport := util.ConstructSvcPortString(newNamespace, service, port)

				g.Ep.RedisClient.DBOneSAdd("temp_"+hostPath, svcport)
				m["temp_"+hostPath] = hostPath

				if newSnippetErr == nil {
					g.Ep.RedisClient.DBOneSAdd("temp_"+hostPath, nameversion)
				}
			}

		}
	}

	for key, value := range m {
		g.Ep.RedisClient.DBOneSUnionStore(value, key)
		g.Ep.RedisClient.DBOneDel(key)
	}
}

// Delete for EventHandler
func (g *IgHandler) Delete(obj interface{}) {
	log.Printf("In INGRESS_HANDLER DELETE %#v \n", obj)
	g.delete(obj)
	g.Ep.RedisClient.PrintAllKeys()
}

// Helper for Deletes
func (g *IgHandler) delete(obj interface{}) {
	ingressObj, ok := obj.(*nv1.Ingress)
	if !ok {
		log.Println("In HandlerIngress Delete; cannot cast to *nv1.Ingress")
		return
	}

	namespace := ingressObj.GetNamespace()
	// v1.18 ingress class name field in ingress object
	//ingressClass, _ := util.ExtractIngressClass(ingressObj.GetAnnotations())
	ingressClass, _ := util.ExtractIngressClassName(obj)
	if !g.Ep.NsManager.IncludeNamespace(namespace) || !g.Ep.ATSManager.IncludeIngressClass(ingressClass) {
		log.Println("Namespace not included or Ingress Class not matched")
		return
	}

	name := ingressObj.GetName()
	version := ingressObj.GetResourceVersion()
	nameversion := util.ConstructNameVersionString(namespace, name, version)

	_, snippetErr := util.ExtractServerSnippet(ingressObj.GetAnnotations())

	if ingressObj.Spec.DefaultBackend != nil {
		host := "*"
		scheme := "http"
		path := "/"
		pathType := nv1.PathTypePrefix
		hostPath := util.ConstructHostPathString(scheme, host, path, pathType)
		service := ingressObj.Spec.DefaultBackend.Service.Name
		port := strconv.Itoa(int(ingressObj.Spec.DefaultBackend.Service.Port.Number))
		svcport := util.ConstructSvcPortString(namespace, service, port)

		g.Ep.RedisClient.DBOneSRem(hostPath, svcport)

		if snippetErr == nil {
			g.Ep.RedisClient.DBOneSRem(hostPath, nameversion)
		}

		scheme = "https"
		hostPath = util.ConstructHostPathString(scheme, host, path, pathType)

		g.Ep.RedisClient.DBOneSRem(hostPath, svcport)

		if snippetErr == nil {
			g.Ep.RedisClient.DBOneSRem(hostPath, nameversion)
		}
	}

	tlsHosts := make(map[string]string)

	for _, ingressTLS := range ingressObj.Spec.TLS {
		for _, tlsHost := range ingressTLS.Hosts {
			tlsHosts[tlsHost] = "1"
		}
	}

	for _, ingressRule := range ingressObj.Spec.Rules {
		host := ingressRule.Host
		if host == "" {
			host = "*"
		}
		scheme := "http"
		if _, ok := tlsHosts[host]; ok {
			scheme = "https"
		}

		for _, httpPath := range ingressRule.HTTP.Paths {

			path := httpPath.Path
			pathType := *httpPath.PathType
			hostPath := util.ConstructHostPathString(scheme, host, path, pathType)
			service := httpPath.Backend.Service.Name
			port := strconv.Itoa(int(httpPath.Backend.Service.Port.Number))
			svcport := util.ConstructSvcPortString(namespace, service, port)

			g.Ep.RedisClient.DBOneSRem(hostPath, svcport)

			if snippetErr == nil {
				g.Ep.RedisClient.DBOneSRem(hostPath, nameversion)
			}
		}

	}

	// release this Ingress's host claims (no-op for hosts owned elsewhere)
	for _, host := range ingressHosts(ingressObj) {
		g.releaseHost(host, namespace)
	}
}

// GetResourceName returns the resource name
func (g *IgHandler) GetResourceName() string {
	return g.ResourceName
}
