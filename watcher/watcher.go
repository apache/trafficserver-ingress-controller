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

	nv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/fields"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/proxy"
)

// FIXME: watching all namespace does not work...

// Watcher stores all essential information to act on HostGroups
type Watcher struct {
	Cs           kubernetes.Interface
	ATSNamespace string
	ResyncPeriod time.Duration
	Ep           *endpoint.Endpoint
	StopChan     chan struct{}
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

	sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    h.Add,
		UpdateFunc: h.Update,
		DeleteFunc: h.Delete,
	})

	go sharedInformer.Run(w.StopChan) // new thread

	if !cache.WaitForCacheSync(w.StopChan, sharedInformer.HasSynced) {
		s := fmt.Sprintf("Timed out waiting for %s caches to sync", h.GetResourceName())
		utilruntime.HandleError(fmt.Errorf(s))
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

		sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    h.Add,
			UpdateFunc: h.Update,
			DeleteFunc: h.Delete,
		})

		go sharedInformer.Run(w.StopChan)

		syncFuncs[i] = sharedInformer.HasSynced
	}
	if !cache.WaitForCacheSync(w.StopChan, syncFuncs...) {
		s := fmt.Sprintf("Timed out waiting for %s caches to sync", h.GetResourceName())
		utilruntime.HandleError(fmt.Errorf(s))
		return errors.New(s)
	}
	return nil
}
