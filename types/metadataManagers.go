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

package types

import (
	"encoding/json"
	"sync"

	"ingress-ats/util"

	set "github.com/deckarep/golang-set"
)

//----------------- nsSvc ---------------------------------

// map[namespace]map[svc][]PathPtr
type nsSvc struct {
	// ns -> []svc -> []*path
	NsToSvc map[string]*svcPaths
	mux     sync.RWMutex
}

// newNsSvc creates a new empty nsSvc
func newNsSvc() *nsSvc {
	return &nsSvc{
		NsToSvc: make(map[string]*svcPaths),
	}
}

// SetNamespaceSvcPaths adds or sets a path/svc within namespace
func (n *nsSvc) SetNamespaceSvcPaths(namespace string, pathPtr *Path) {
	n.mux.Lock()
	if svcPathsPtr, found := n.NsToSvc[namespace]; found {
		svcPathsPtr.AddSvcPath(pathPtr)
	} else {
		newSvcPaths := newSvcPaths()
		newSvcPaths.AddSvcPath(pathPtr)
		n.NsToSvc[namespace] = newSvcPaths
	}
	n.mux.Unlock()
}

// DelNamespaceSvcPath deletes a Path ptr from namespace
func (n *nsSvc) DelNamespaceSvcPath(pathPtr *Path) {
	n.mux.Lock()
	defer n.mux.Unlock()
	namespace := pathPtr.GetNamespace()
	if svcPathsPtr, found := n.NsToSvc[namespace]; found {
		deletedSvc := svcPathsPtr.DelSvcPath(pathPtr)
		if deletedSvc && svcPathsPtr.NumSvc() == 0 {
			// when ns has no ingress defined deployed service
			delete(n.NsToSvc, namespace)
		}
	}
}

// NumNamespace returns the number of namespaces being managed
func (n *nsSvc) NumNamespace() int {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return len(n.NsToSvc)
}

// HasSvc returns true if namespace exists, and svc exists in
// same namespace
func (n *nsSvc) HasSvc(namespace, svcName string) bool {
	n.mux.RLock()
	defer n.mux.RUnlock()
	if svcPaths, found := n.NsToSvc[namespace]; found {
		return svcPaths.Has(svcName)
	}
	return false
}

// NumHostPathInNamespace returns number of paths of hostName are in namespace
// TODO: can add more metadata in the future to make this faster
func (n *nsSvc) NoHostPathInNamespace(hostName, pathName, namespace string) bool {
	res := 0
	n.mux.RLock()
	svcPathsPtr := n.NsToSvc[namespace]
	n.mux.RUnlock()
	svcPathsPtr.rLock()
	for _, pathSet := range svcPathsPtr.StoPs {
		for path := range pathSet.Iter() {
			pathPtr := path.(*Path)
			if pathPtr.GetHostName() == hostName &&
				pathPtr.GetPathName() == pathName {
				res++
			}
		}
	}
	svcPathsPtr.rUnlock()
	return res == 0
}

// Iter returns a rangable channel of all Paths using svc, within namespace
func (n *nsSvc) Iter(namespace, svc string) <-chan interface{} {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.NsToSvc[namespace].Iter(svc)
}

func (n *nsSvc) String() string {
	n.mux.RLock()
	defer n.mux.RUnlock()
	marshalled, _ := json.Marshal(n)
	return util.FmtMarshalled(marshalled)
}

//----------------- svcPaths ------------------------------

// serviceName -> []PathPtr
type svcPaths struct {
	StoPs map[string]set.Set `json:"StoPs"` // svcName -> []Pathptr
	mux   sync.RWMutex
}

// newSvcPaths returns a new empty SvcPaths struct ptr
func newSvcPaths() *svcPaths {
	return &svcPaths{StoPs: make(map[string]set.Set)}
}

func (s *svcPaths) rLock() {
	s.mux.RLock()
}

func (s *svcPaths) rUnlock() {
	s.mux.RUnlock()
}

// checkAddSvc adds/initialize svc if not stored
func (s *svcPaths) checkAddSvc(svc string) {
	s.mux.Lock()
	if _, found := s.StoPs[svc]; !found {
		s.StoPs[svc] = set.NewSet()
	}
	s.mux.Unlock()
}

// AddSvcPath adds a Mapping svc --> PathPtr safely
func (s *svcPaths) AddSvcPath(pathPtr *Path) {
	// path cannot be updated during this operation
	pathPtr.RLock()
	s.checkAddSvc(pathPtr.ServiceName)
	// v ^ order here cannot switch!
	s.mux.RLock()
	s.StoPs[pathPtr.ServiceName].Add(pathPtr)
	s.mux.RUnlock()
	pathPtr.RUnlock()
}

// DelSvcPath deletes a path ptr associated with svc
// returns true if svc no longer referenced by any path/backend and is deleted
func (s *svcPaths) DelSvcPath(pathPtr *Path) (deletedSvc bool) {
	s.mux.Lock()
	pathPtr.RLock() // path cannot be updated during this operation
	defer pathPtr.RUnlock()
	defer s.mux.Unlock()
	svc := pathPtr.ServiceName
	if _, found := s.StoPs[svc]; found {
		s.StoPs[svc].Remove(pathPtr)
		// if svc no longer needed by any backend
		if s.StoPs[svc].Cardinality() == 0 {
			delete(s.StoPs, svc)
			return true
		}
	}
	return false

}

// Iter returns a channel of path ptrs associated with svc
// this is meant to be used with "range"
func (s *svcPaths) Iter(svc string) <-chan interface{} {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.StoPs[svc].Iter() // this itself is threadsafe
}

// Has returns whether the svc is stored
func (s *svcPaths) Has(svc string) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	_, found := s.StoPs[svc]
	return found
}

// NumSvc returns number of svcs stored
func (s *svcPaths) NumSvc() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.StoPs)
}

func (s *svcPaths) String() string {
	s.mux.RLock()
	defer s.mux.RUnlock()
	marshalled, _ := json.Marshal(s)
	return util.FmtMarshalled(marshalled)
}

//----------------- hostNamespaces ------------------------

// hostName -> []namespace
type hostNamespaces struct {
	HtoN map[string]set.Set
	mux  sync.RWMutex
}

// newHostNamespaces returns a new empty hostNamespaces struct
func newHostNamespaces() *hostNamespaces {
	return &hostNamespaces{
		HtoN: make(map[string]set.Set),
	}
}

// checkInit checks of hostName is initialized in map
func (h *hostNamespaces) checkInit(hostName string) {
	h.mux.Lock()
	if _, found := h.HtoN[hostName]; !found {
		h.HtoN[hostName] = set.NewSet()
	}
	h.mux.Unlock()
}

// AddNamespace adds namespace to hostname's set of namespaces
// that it belongs to
func (h *hostNamespaces) AddNamespace(hostName, namespace string) {
	h.checkInit(hostName)
	h.mux.RLock()
	h.HtoN[hostName].Add(namespace)
	h.mux.RUnlock()
}

// DelNamespace deletes namespace from hostname's set of namespaces
// if host no longer belongs in any namespace, function will clear hostname
// from its map and return true.
func (h *hostNamespaces) DelNamespace(hostName, namespace string) (deletedHost bool) {
	h.mux.Lock()
	defer h.mux.Unlock()
	_, found := h.HtoN[hostName]
	if found {
		h.HtoN[hostName].Remove(namespace)
		if h.HtoN[hostName].Cardinality() == 0 {
			delete(h.HtoN, hostName)
			return true
		}
	}
	return false
}

// hostPathInNamespace returns true if host has a path in namespace
func (h *hostNamespaces) hostPathInNamespace(hostName, namespace string) bool {
	h.mux.RLock()
	defer h.mux.RUnlock()
	_, found := h.HtoN[hostName]
	if found {
		return h.HtoN[hostName].Contains(namespace)
	}
	return false
}

// hostOnlyInNamespace returns true if host only has path(s) in namespace
// and nowhere else
func (h *hostNamespaces) hostOnlyInNamespace(hostName, namespace string) bool {
	h.mux.RLock()
	defer h.mux.RUnlock()
	_, found := h.HtoN[hostName]
	if !found {
		// returning false here because host might
		// very well be deleted by another thread before this function
		// got ran; in which case, return false so caller don't do
		// anything that might be dangerous
		return false
	}
	return h.HtoN[hostName].Cardinality() == 1 &&
		h.HtoN[hostName].Contains(namespace)

}
