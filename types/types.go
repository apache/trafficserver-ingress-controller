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
	"log"
	"sync"

	"ingress-ats/util"
	// set.NewSet() is threadsafe from this package
	set "github.com/deckarep/golang-set"
)

// ControllerConfig is the outmost struct storing everything
// as part of the outputs
type ControllerConfig struct {
	// Annotations []SSPair     `json:"annotations"`
	ConfigGroup *ConfigGroup `json:"configgroup"`
	HostGroup   *HostGroup   `json:"hostgroup"`
}

//---------------------------------------------------------
//----------------- ConfigGroup ---------------------------
//---------------------------------------------------------

// ConfigGroup stores all ConfigMap settings for ATS
type ConfigGroup struct {
	ConfigMap *configMap `json:"configmaps"`
}

// NewConfigGroup returns an empty new ConfigGroup struct
func NewConfigGroup() *ConfigGroup {
	return &ConfigGroup{
		ConfigMap: newConfigMap(),
	}
}

//----------------- ConfigMap -----------------------------

// ConfigMap stores one Configmap
type configMap struct {
	Annotations map[string]string `json:"annotations"`
	Data        map[string]string `json:"data"`
	mux         sync.RWMutex
}

type filter func(string) bool

// this is possible because map in Go are references
func (c *configMap) load(dest, src map[string]string, fn filter) {
	c.mux.Lock()
	for k, v := range src {
		if fn(k) {
			dest[k] = v
		}
	}
	c.mux.Unlock()
}

// LoadAnnotations loads everything in input map safely
func (c *configMap) LoadAnnotations(input map[string]string, fn filter) {
	c.load(c.Annotations, input, fn)
}

// LoadData loads everything in input map safely
func (c *configMap) LoadData(input map[string]string, fn filter) {
	c.load(c.Data, input, fn)
}

// DelFromData deletes a key from Data
func (c *configMap) DelFromData(key string) {
	c.mux.Lock()
	delete(c.Data, key)
	c.mux.Unlock()
}

// HasKeyVal returns whether this ConfigMap contains a key mapped to val
func (c *configMap) HasKeyVal(key, val string) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if v, found := c.Data[key]; found {
		return v == val
	}
	return false
}

// SetData does add and update
func (c *configMap) SetData(k, v string) {
	c.mux.Lock()
	c.Data[k] = v
	c.mux.Unlock()
}

// String is toString
func (c *configMap) String() string {
	c.mux.RLock()
	defer c.mux.RUnlock()
	marshalled, _ := json.Marshal(c)
	return util.FmtMarshalled(marshalled)
}

// NewConfigMap creates a new empty ConfigMap struct
func newConfigMap() *configMap {
	return &configMap{
		Annotations: make(map[string]string),
		Data:        make(map[string]string)}
}

//---------------------------------------------------------
//----------------- HostGroup -----------------------------
//---------------------------------------------------------

// TODO: HostGroup methods should orchestrate everything
// under it, as well as updating both Hosts, and ServiceMgr

// HostGroup is a collection of Hosts
type HostGroup struct {
	Hosts      map[string]*Host `json:"hosts"` // different Hosts e.g. test.akomljen.com
	ServiceMgr *nsSvc           `json:"-"`
	HostNsMgr  *hostNamespaces  `json:"-"`
	mux        sync.RWMutex
}

// NewHostGroup returns an empty HostGroup struct
func NewHostGroup() *HostGroup {
	return &HostGroup{
		Hosts:      make(map[string]*Host),
		ServiceMgr: newNsSvc(),
		HostNsMgr:  newHostNamespaces(),
	}
}

// hasHost returns true if hostName is already stored in HostGroup
func (h *HostGroup) hasHost(hostName string) bool {
	h.mux.RLock()
	defer h.mux.RUnlock()
	_, found := h.Hosts[hostName]
	return found
}

// AddHost adds a new Host -> HostPtr to HostGroup
func (h *HostGroup) AddHost(hostName string, hostPtr *Host) {
	if h.hasHost(hostName) {
		log.Panicf("HostGroup::AddHost(%s) existing host", hostName)
	}
	h.mux.Lock()
	h.Hosts[hostName] = hostPtr
	h.mux.Unlock()
}

// DelHost safely deletes a hostName
func (h *HostGroup) DelHost(hostName string) {
	h.mux.RLock()
	targetHost := h.Hosts[hostName]
	h.mux.RUnlock()
	targetHost.lock()
	for _, pathPtr := range targetHost.Paths {
		h.ServiceMgr.DelNamespaceSvcPath(pathPtr)
	}
	targetHost.unlock()
	h.mux.Lock()
	delete(h.Hosts, hostName)
	h.mux.Unlock()
}

// GetHost returns Host of hostName
func (h *HostGroup) GetHost(hostName string) *Host {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.Hosts[hostName]
}

// HostPathInNamespace returns true if host belongs in namespace
func (h *HostGroup) HostPathInNamespace(hostName, namespace string) bool {
	return h.HostNsMgr.hostPathInNamespace(hostName, namespace)
}

// HostOnlyInNamespace returns true if host only has path(s) in namespace
// and nowhere else
func (h *HostGroup) HostOnlyInNamespace(hostName, namespace string) bool {
	return h.HostNsMgr.hostOnlyInNamespace(hostName, namespace)
}

//----------------- Host ----------------------------------

// Host stores a host name to ALL of its paths
// From Ingress Resource
// NOTE: if too many paths exist e.g. x/  x/y  x/y/z etc. maybe implement trees
type Host struct { // a single Host
	HostName string           `json:"hostname"` // host name
	Paths    map[string]*Path `json:"paths"`    // a list of paths
	mux      sync.RWMutex
}

// NewHost returns a new empty Host struct
func NewHost(hostName string) *Host {
	return &Host{
		HostName: hostName,
		Paths:    make(map[string]*Path),
	}
}

// lock locks up Host
func (h *Host) lock() {
	h.mux.Lock()
}

// unlock once done
func (h *Host) unlock() {
	h.mux.Unlock()
}

// hasPath return true if path exists under Host
func (h *Host) hasPath(pathName string) bool {
	h.mux.RLock()
	defer h.mux.RUnlock()
	_, found := h.Paths[pathName]
	return found
}

// GetPath safely returns the Path ptr under pathName
func (h *Host) GetPath(pathName string) *Path {
	h.mux.RLock()
	defer h.mux.RUnlock()
	return h.Paths[pathName]
}

// AddPath adds a new pathName -> pathPtr
func (h *Host) AddPath(pathName string, pathPtr *Path) {
	if h.hasPath(pathName) {
		log.Panicf("Host::AddPath(%s, pathPtr) already exists", pathName)
	}
	h.mux.Lock()
	h.Paths[pathName] = pathPtr
	h.mux.Unlock()
}

// DelPath safely deletes a pathName
func (h *Host) DelPath(pathName string) {
	h.mux.Lock()
	delete(h.Paths, pathName)
	h.mux.Unlock()
}

// HasDuplicatePath returns true if pathName is defined in a namespace that is
// *different* from newNamespace
func (h *Host) HasDuplicatePath(pathName, newNamespace string) bool {
	h.mux.RLock()
	defer h.mux.RUnlock()
	if pathPtr, found := h.Paths[pathName]; found {
		return pathPtr.GetNamespace() != newNamespace
	}
	return false
}

//----------------- Path ----------------------------------

// Path is a specific path e.g. /api that stores some request data
// as well as services associated with the path.
// From Ingress Resource
type Path struct { // A single Path
	HostName    string  `json:"hostname"`    // host name of this path
	PathName    string  `json:"pathname"`    // path name
	Namespace   string  `json:"namespace"`   // namespace this path is in
	ServiceName string  `json:"servicename"` // service name associated with path name
	ServicePort string  `json:"serviceport"` // port of referenced service
	Server      *Server `json:"server"`      // A list of services
	mux         sync.RWMutex
}

// NewPath constructs and returns a ptr to a new path struct
func NewPath(hostName, pathName, namespace, serviceName, servicePort string,
	server *Server) *Path {
	return &Path{
		HostName:    hostName,
		PathName:    pathName,
		Namespace:   namespace,
		ServiceName: serviceName,
		ServicePort: servicePort,
		Server:      server,
	}
}

func (p *Path) String() string {
	p.mux.RLock()
	defer p.mux.RUnlock()
	marshalled, _ := json.Marshal(p)
	return util.FmtMarshalled(marshalled)
}

// RLock used by other struct
func (p *Path) RLock() {
	p.mux.RLock()
}

// RUnlock when done
func (p *Path) RUnlock() {
	p.mux.RUnlock()
}

// GetHostName returns HostName of this Path
// this never changes, no locking
func (p *Path) GetHostName() string {
	return p.HostName
}

// GetPathName returns Path Name of this Path
// this never changes, no locking
func (p *Path) GetPathName() string {
	return p.PathName
}

// GetNamespace returns the namespace this Path belongs to
// this never changes, no locking
func (p *Path) GetNamespace() string {
	return p.Namespace
}

// GetServiceName returns serviceName of this Path
func (p *Path) GetServiceName() string {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.ServiceName
}

// SetService safely sets the serviceName of this Path
// this also includes updating the server!
func (p *Path) SetService(serviceName string, server *Server) {
	p.mux.Lock()
	p.ServiceName = serviceName
	p.Server = server
	p.mux.Unlock()
}

// GetServicePort returns servicePort of this Path
func (p *Path) GetServicePort() string {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.ServicePort
}

// SetServicePort safely sets the servicePort this Path
func (p *Path) SetServicePort(servicePort string) {
	p.mux.Lock()
	p.ServicePort = servicePort
	p.mux.Unlock()
}

// InNamespace returns true if Path belongs in namespace
// this never changes, no locking
func (p *Path) InNamespace(namespace string) bool {
	return p.Namespace == namespace
}

//----------------- Server --------------------------------

// Server stores essential info of each service and how to reach it.
// From Endpoint Resource
type Server struct {
	IPAddresses set.Set `json:"ipaddresses"` // []string IP address IPV4 or IPV6
	Ports       set.Set `json:"ports"`       // []Ports port number of the pod
}

// NewServer returns a new emtpy server struct
func NewServer() *Server {
	return &Server{
		IPAddresses: set.NewSet(),
		Ports:       set.NewSet(),
	}
}

// Port stores job of each port number
// From Endpoint Resource
type Port struct {
	Name     string `json:"name"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"` // TCP, UDP, HTTP etc
}

//---------------------------------------------------------
//-------------- Helper [unused] --------------------------
//---------------------------------------------------------

// SSPair stores key val String pairs of one annotation
// Annotations is an unstructured key value map stored with a resource that may be
// set by external tools to store and retrieve arbitrary metadata. They are not
// queryable and should be preserved when modifying objects.
// More info: http://kubernetes.io/docs/user-guide/annotations
type SSPair struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

// CreateSSPairs constructs an array of SSPair using a string string map and
// a filter function
func CreateSSPairs(m map[string]string, filter func(string) bool) []SSPair {
	if m != nil && len(m) > 0 {
		var res []SSPair
		for k, v := range m {
			if filter(k) {
				res = append(res, SSPair{
					Key: k,
					Val: v,
				})
			}
		}
		return res
	}
	return nil
}
