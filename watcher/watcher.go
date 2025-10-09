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
	"errors"
	"fmt"
	"log"
	"time"

	v1 "k8s.io/api/core/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/proxy"
	nv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
)

const CACHE_PATH string = "/opt/ats/etc/trafficserver/cache.config"

// FIXME: watching all namespace does not work...

// Watcher stores all essential information to act on HostGroups
type Watcher struct {
	Cs            kubernetes.Interface
	DynamicClient dynamic.Interface
	ATSNamespace  string
	ResyncPeriod  time.Duration
	Ep            *endpoint.Endpoint
	StopChan      chan struct{}
}

// EventHandler interface defines the 3 required methods to implement for watchers
type EventHandler interface {
	Add(obj interface{})
	Update(obj, newObj interface{})
	Delete(obj interface{})
	GetResourceName() string // EventHandler should store the ResourceName e.g. ingresses, endpoints...
}

// Watch creates necessary threads to watch over resources
func (w *Watcher) Watch() error {
	//================= Watch for Ingress ==================
	igHandler := IgHandler{"ingresses", w.Ep}
	igListWatch := cache.NewListWatchFromClient(w.Cs.NetworkingV1().RESTClient(), igHandler.GetResourceName(), v1.NamespaceAll, fields.Everything())
	err := w.allNamespacesWatchFor(&igHandler, w.Cs.NetworkingV1().RESTClient(),
		fields.Everything(), &nv1.Ingress{}, w.ResyncPeriod, igListWatch)
	if err != nil {
		return err
	}
	//================= Watch for Endpoints =================
	epHandler := EpHandler{"endpoints", w.Ep}
	epListWatch := cache.NewListWatchFromClient(w.Cs.CoreV1().RESTClient(), epHandler.GetResourceName(), v1.NamespaceAll, fields.Everything())
	err = w.allNamespacesWatchFor(&epHandler, w.Cs.CoreV1().RESTClient(),
		fields.Everything(), &v1.Endpoints{}, w.ResyncPeriod, epListWatch)
	if err != nil {
		return err
	}
	//================= Watch for ConfigMaps =================
	cmHandler := CMHandler{"configmaps", w.Ep}
	targetNs := make([]string, 1)
	targetNs[0] = w.Ep.ATSManager.(*proxy.ATSManager).Namespace
	err = w.inNamespacesWatchFor(&cmHandler, w.Cs.CoreV1().RESTClient(),
		targetNs, fields.Everything(), &v1.ConfigMap{}, w.ResyncPeriod)
	if err != nil {
		return err
	}

	log.Println("calling the Watch Ats Caching Policy function")
	if err := w.WatchAtsCachingPolicy(CACHE_PATH); err != nil {
		return err
	}
	return nil
}

func (w *Watcher) allNamespacesWatchFor(h EventHandler, c cache.Getter,
	fieldSelector fields.Selector, objType pkgruntime.Object,
	resyncPeriod time.Duration, listerWatcher cache.ListerWatcher) error {

	factory := informers.NewSharedInformerFactory(w.Cs, resyncPeriod)
	var sharedInformer cache.SharedIndexInformer
	switch objType.(type) {
	case *v1.Endpoints:
		sharedInformer = factory.Core().V1().Endpoints().Informer()
	case *nv1.Ingress:
		sharedInformer = factory.Networking().V1().Ingresses().Informer()
	}

	_, err := sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    h.Add,
		UpdateFunc: h.Update,
		DeleteFunc: h.Delete,
	})
	if err != nil {
		return err
	}

	go sharedInformer.Run(w.StopChan) // new thread

	if !cache.WaitForCacheSync(w.StopChan, sharedInformer.HasSynced) {
		s := fmt.Sprintf("Timed out waiting for %s caches to sync", h.GetResourceName())
		utilruntime.HandleError(errors.New(s))
		return errors.New(s)
	}
	return nil
}

// This is meant to make it easier to add resource watchers on resources that
// span multiple namespaces
func (w *Watcher) inNamespacesWatchFor(h EventHandler, c cache.Getter,
	namespaces []string, fieldSelector fields.Selector, objType pkgruntime.Object,
	resyncPeriod time.Duration) error {
	if len(namespaces) == 0 {
		log.Panicln("inNamespacesWatchFor must have at least 1 namespace")
	}
	syncFuncs := make([]cache.InformerSynced, len(namespaces))
	for i, ns := range namespaces {
		factory := informers.NewSharedInformerFactoryWithOptions(w.Cs, resyncPeriod, informers.WithNamespace(ns))

		var sharedInformer cache.SharedIndexInformer
		switch objType.(type) {
		case *v1.Endpoints:
			sharedInformer = factory.Core().V1().Endpoints().Informer()
		case *nv1.Ingress:
			sharedInformer = factory.Networking().V1().Ingresses().Informer()
		case *v1.ConfigMap:
			sharedInformer = factory.Core().V1().ConfigMaps().Informer()
		}

		_, err := sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    h.Add,
			UpdateFunc: h.Update,
			DeleteFunc: h.Delete,
		})
		if err != nil {
			return err
		}

		go sharedInformer.Run(w.StopChan)

		syncFuncs[i] = sharedInformer.HasSynced
	}
	if !cache.WaitForCacheSync(w.StopChan, syncFuncs...) {
		s := fmt.Sprintf("Timed out waiting for %s caches to sync", h.GetResourceName())
		utilruntime.HandleError(errors.New(s))
		return errors.New(s)
	}
	return nil
}

func (w *Watcher) WatchAtsCachingPolicy(path string) error {
	gvr := schema.GroupVersionResource{Group: "k8s.trafficserver.apache.com", Version: "v1alpha1", Resource: "atscachingpolicies"}
	dynamicFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(w.DynamicClient, w.ResyncPeriod, metav1.NamespaceAll, nil)
	informer := dynamicFactory.ForResource(gvr).Informer()
	cachehandler := NewAtsCacheHandler("atscaching", w.Ep, path)
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cachehandler.Add,
		UpdateFunc: cachehandler.Update,
		DeleteFunc: cachehandler.Delete,
	})

	if err != nil {
		return fmt.Errorf("failed to add event handler: %v\n", err)
	}

	go informer.Run(w.StopChan)
	if !cache.WaitForCacheSync(w.StopChan, informer.HasSynced) {
		return fmt.Errorf("failed to sync ATSCachingPolicy informer")
	}
	log.Println("ATSCachingPolicy informer running and synced")
	return nil
}
