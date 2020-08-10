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
	ep "ingress-ats/endpoint"
	"ingress-ats/namespace"
	"ingress-ats/proxy"
	"ingress-ats/redis"
	"log"
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdd_BasicConfigMap(t *testing.T) {
	cmHandler := createExampleCMHandler()
	exampleConfigMap := createExampleConfigMap()

	cmHandler.Add(&exampleConfigMap)

	rEnabled, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_enabled")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rEnabled, "1") {
		t.Errorf("returned \n%s,  but expected \n%s", rEnabled, "1")
	}

	rInterval, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_interval_sec")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rInterval, "3000") {
		t.Errorf("returned \n%s,  but expected \n%s", rInterval, "3000")
	}

	threshold, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.restart.active_client_threshold")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(threshold, "0") {
		t.Errorf("returned \n%s,  but expected \n%s", threshold, "0")
	}

}

func TestShouldNotAdd_BasicConfigMap(t *testing.T) {
	cmHandler := createExampleCMHandler()
	exampleConfigMap := createExampleConfigMap()

	exampleConfigMap.Annotations = map[string]string{
		"ats-configmap": "false",
	}

	cmHandler.Add(&exampleConfigMap)

	rEnabled, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_enabled")

	if err == nil {
		t.Errorf("Should not have executed. Instead gives %s", rEnabled)
	}
}

func TestUpdate_BasicConfigMap(t *testing.T) {
	cmHandler := createExampleCMHandler()
	exampleConfigMap := createExampleConfigMap()
	exampleConfigMap.Data["proxy.config.output.logfile.rolling_interval_sec"] = "2000"

	cmHandler.update(&exampleConfigMap)

	rEnabled, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_enabled")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rEnabled, "1") {
		t.Errorf("returned \n%s,  but expected \n%s", rEnabled, "1")
	}

	rInterval, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_interval_sec")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rInterval, "2000") {
		t.Errorf("returned \n%s,  but expected \n%s", rInterval, "2000")
	}

	threshold, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.restart.active_client_threshold")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(threshold, "0") {
		t.Errorf("returned \n%s,  but expected \n%s", threshold, "0")
	}

}

func createExampleConfigMap() v1.ConfigMap {
	exampleConfigMap := v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver-test-2",
			Annotations: map[string]string{
				"ats-configmap": "true",
			},
		},
		Data: map[string]string{
			"proxy.config.output.logfile.rolling_enabled":      "1",
			"proxy.config.output.logfile.rolling_interval_sec": "3000",
			"proxy.config.restart.active_client_threshold":     "0",
		},
	}

	return exampleConfigMap
}

func createExampleCMHandler() CMHandler {
	exampleEndpoint := createExampleEndpointWithFakeATS()
	cmHandler := CMHandler{"configmap", &exampleEndpoint}

	return cmHandler
}

func createExampleEndpointWithFakeATS() ep.Endpoint {
	rClient, err := redis.InitForTesting()
	if err != nil {
		log.Panicln("Redis Error: ", err)
	}

	namespaceMap := make(map[string]bool)
	ignoreNamespaceMap := make(map[string]bool)

	nsManager := namespace.NsManager{
		NamespaceMap:       namespaceMap,
		IgnoreNamespaceMap: ignoreNamespaceMap,
	}

	nsManager.Init()

	exampleEndpoint := ep.Endpoint{
		RedisClient: rClient,
		ATSManager: &proxy.FakeATSManager{
			Namespace:    "default",
			IngressClass: "",
			Config:       make(map[string]string),
		},
		NsManager: &nsManager,
	}

	return exampleEndpoint
}
