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

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/util/workqueue"

	_ "k8s.io/api/networking/v1"

	ep "github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/namespace"
	"github.com/apache/trafficserver-ingress-controller/proxy"
	"github.com/apache/trafficserver-ingress-controller/redis"
	w "github.com/apache/trafficserver-ingress-controller/watcher"
)

var (
	apiServer          = flag.String("apiServer", "http://notfound.com/", "The kubernetes api server. It should be set if useKubeConfig is set to false. Setting to a dummy value to prevent accidental usage.")
	useKubeConfig      = flag.Bool("useKubeConfig", false, "Set to true to use kube config passed in kubeconfig arg.")
	useInClusterConfig = flag.Bool("useInClusterConfig", true, "Set to false to opt out incluster config passed in kubeconfig arg.")
	kubeconfig         = flag.String("kubeconfig", "/usr/local/etc/k8s/k8s.config", "Absolute path to the kubeconfig file. Only works if useKubeConfig is set to true.")

	certFilePath = flag.String("certFilePath", "/etc/pki/tls/certs/kube-router.pem", "Absolute path to kuberouter user cert file.")
	keyFilePath  = flag.String("keyFilePath", "/etc/pki/tls/private/kube-router-key.pem", "Absolute path to kuberouter user key file.")
	caFilePath   = flag.String("caFilePath", "/etc/pki/tls/certs/ca.pem", "Absolute path to k8s cluster ca file.")

	namespaces       = flag.String("namespaces", namespace.ALL, "Comma separated list of namespaces to watch for ingress and endpoints.")
	ignoreNamespaces = flag.String("ignoreNamespaces", "", "Comma separated list of namespaces to ignore for ingress and endpoints.")

	atsNamespace    = flag.String("atsNamespace", "default", "Name of Namespace the ATS pod resides.")
	atsIngressClass = flag.String("atsIngressClass", "", "Ingress Class of Ingress object that ATS will retrieve routing info from")

	resyncPeriod = flag.Duration("resyncPeriod", 0*time.Second, "Resync period for the cache of informer")
)

func init() {
	flag.Parse()
}

func main() {
	var (
		config                           *rest.Config
		err                              error
		namespaceMap, ignoreNamespaceMap map[string]bool
	)

	if *atsNamespace == "" {
		log.Panicln("Not all required args given.")
	}

	namespaceMap = make(map[string]bool)

	// default namespace to "all"
	if *namespaces != namespace.ALL {
		namespaceList := strings.Split(strings.Replace(strings.ToLower(*namespaces), " ", "", -1), ",")
		for _, namespace := range namespaceList {
			if namespace != "" {
				namespaceMap[namespace] = true
			}
		}
	}

	ignoreNamespaceMap = make(map[string]bool)

	if *ignoreNamespaces != "" {
		ignoreNamespaceList := strings.Split(strings.Replace(strings.ToLower(*ignoreNamespaces), " ", "", -1), ",")
		for _, namespace := range ignoreNamespaceList {
			if namespace != "" {
				ignoreNamespaceMap[namespace] = true
			}
		}
	}

	if *useKubeConfig {
		log.Println("Read config from ", *kubeconfig)
		/* For running outside of the cluster
		uses the current context in kubeconfig */
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Panicln(err.Error())
		}
		/* for running inside the cluster */
	} else if *useInClusterConfig {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Panicln("failed to create InClusterConfig: " + err.Error())
		}
	} else {
		/* create config and set necessary parameters */
		config = &rest.Config{}
		config.Host = *apiServer
		config.CertFile = *certFilePath
		config.KeyFile = *keyFilePath
		config.CAFile = *caFilePath
	}

	/* creates the clientset */
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicln(err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Panicln(err.Error())
	}

	stopChan := make(chan struct{})

	// ------------ Resolving Namespaces --------------------------------------
	nsManager := namespace.NsManager{
		NamespaceMap:       namespaceMap,
		IgnoreNamespaceMap: ignoreNamespaceMap,
	}

	nsManager.Init()

	//------------ Setting up Redis in memory Datastructure -------------------
	rClient, err := redis.Init()
	if err != nil {
		log.Panicln("Redis Error: ", err)
	}

	// ALL services must be using CORE V1 API
	endpoint := ep.Endpoint{
		RedisClient: rClient,
		ATSManager:  &proxy.ATSManager{Namespace: *atsNamespace, IngressClass: *atsIngressClass},
		NsManager:   &nsManager,
	}

	watcher := w.Watcher{
		Cs:            clientset,
		DynamicClient: dynamicClient,
		ATSNamespace:  *atsNamespace,
		ResyncPeriod:  *resyncPeriod,
		Ep:            &endpoint,
		StopChan:      stopChan,
	}

	err = watcher.Watch()
	if err != nil {
		log.Panicln("Error received from watcher.Watch() :", err)
	}

	/* Program termination */
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for range signalChan {
		log.Println("Shutdown signal received, exiting...")
		close(stopChan)
		os.Exit(0)
	}
}
