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

	v1beta1 "k8s.io/api/extensions/v1beta1"

	"k8s.io/apimachinery/pkg/fields"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"ingress-ats/endpoint"
	"ingress-ats/proxy"
)

// FIXME: watching all namespace does not work...

// Watcher stores all essential information to act on HostGroups
type Watcher struct {
	Cs           kubernetes.Interface
	ATSNamespace string
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
	igListWatch := cache.NewListWatchFromClient(w.Cs.ExtensionsV1beta1().RESTClient(), igHandler.GetResourceName(), v1.NamespaceAll, fields.Everything())
	err := w.allNamespacesWatchFor(&igHandler, w.Cs.ExtensionsV1beta1().RESTClient(),
		fields.Everything(), &v1beta1.Ingress{}, 0, igListWatch)
	if err != nil {
		return err
	}
	//================= Watch for Endpoints =================
	epHandler := EpHandler{"endpoints", w.Ep}
	epListWatch := cache.NewListWatchFromClient(w.Cs.CoreV1().RESTClient(), epHandler.GetResourceName(), v1.NamespaceAll, fields.Everything())
	err = w.allNamespacesWatchFor(&epHandler, w.Cs.CoreV1().RESTClient(),
		fields.Everything(), &v1.Endpoints{}, 0, epListWatch)
	if err != nil {
		return err
	}
	//================= Watch for ConfigMaps =================
	cmHandler := CMHandler{"configmaps", w.Ep}
	targetNs := make([]string, 1, 1)
	targetNs[0] = w.Ep.ATSManager.(*proxy.ATSManager).Namespace
	err = w.inNamespacesWatchForConfigMaps(&cmHandler, w.Cs.CoreV1().RESTClient(),
		targetNs, fields.Everything(), &v1.ConfigMap{}, 0, w.Cs)
	if err != nil {
		return err
	}
	return nil
}

func (w *Watcher) allNamespacesWatchFor(h EventHandler, c cache.Getter,
	fieldSelector fields.Selector, objType pkgruntime.Object,
	resyncPeriod time.Duration, listerWatcher cache.ListerWatcher) error {
	sharedInformer := cache.NewSharedInformer(listerWatcher, objType, resyncPeriod)

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
func (w *Watcher) inNamespacesWatchForConfigMaps(h EventHandler, c cache.Getter,
	namespaces []string, fieldSelector fields.Selector, objType pkgruntime.Object,
	resyncPeriod time.Duration, clientset kubernetes.Interface) error {
	if len(namespaces) == 0 {
		log.Panic("inNamespacesWatchFor must have at least 1 namespace")
	}
	syncFuncs := make([]cache.InformerSynced, len(namespaces))
	for i, ns := range namespaces {
		factory := informers.NewSharedInformerFactoryWithOptions(clientset, resyncPeriod, informers.WithNamespace(ns))
		cmInfo := factory.Core().V1().ConfigMaps().Informer()

		cmInfo.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    h.Add,
			UpdateFunc: h.Update,
			DeleteFunc: h.Delete,
		})

		go cmInfo.Run(w.StopChan)

		syncFuncs[i] = cmInfo.HasSynced
	}
	if !cache.WaitForCacheSync(w.StopChan, syncFuncs...) {
		s := fmt.Sprintf("Timed out waiting for %s caches to sync", h.GetResourceName())
		utilruntime.HandleError(fmt.Errorf(s))
		return errors.New(s)
	}
	return nil
}

func (w *Watcher) inNamespacesWatchFor(h EventHandler, c cache.Getter,
	namespaces []string, fieldSelector fields.Selector, objType pkgruntime.Object,
	resyncPeriod time.Duration) error {
	if len(namespaces) == 0 {
		log.Panic("inNamespacesWatchFor must have at least 1 namespace")
	}
	syncFuncs := make([]cache.InformerSynced, len(namespaces))
	for i, ns := range namespaces {
		epListWatch := cache.NewListWatchFromClient(c, h.GetResourceName(), ns, fieldSelector)
		sharedInformer := cache.NewSharedInformer(epListWatch, objType, resyncPeriod)

		sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    h.Add,
			UpdateFunc: h.Update,
			DeleteFunc: h.Delete,
		})

		go sharedInformer.Run(w.StopChan) // new thread

		syncFuncs[i] = sharedInformer.HasSynced
	}
	if !cache.WaitForCacheSync(w.StopChan, syncFuncs...) {
		s := fmt.Sprintf("Timed out waiting for %s caches to sync", h.GetResourceName())
		utilruntime.HandleError(fmt.Errorf(s))
		return errors.New(s)
	}
	return nil
}
