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
	"context"
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	fake "k8s.io/client-go/kubernetes/fake"
	framework "k8s.io/client-go/tools/cache/testing"
)

func TestAllNamespacesWatchFor_Add(t *testing.T) {
	w, fc := getTestWatcher()

	epHandler := EpHandler{"endpoints", w.Ep}
	err := w.allNamespacesWatchFor(&epHandler, w.Cs.CoreV1().RESTClient(),
		fields.Everything(), &v1.Endpoints{}, 0, fc)

	if err != nil {
		t.Error(err)
	}

	w.Cs.CoreV1().Endpoints("trafficserver-test-2").Create(context.TODO(), &v1.Endpoints{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver-test-2",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "10.10.1.1",
					},
					{
						IP: "10.10.2.2",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "main",
						Port:     8080,
						Protocol: "TCP",
					},
				},
			},
		},
	}, meta_v1.CreateOptions{})

	time.Sleep(100 * time.Millisecond)

	returnedKeys := w.Ep.RedisClient.GetDefaultDBKeyValues()
	expectedKeys := getExpectedKeysForEndpointAdd()

	if !reflect.DeepEqual(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestAllNamespacesWatchFor_Update(t *testing.T) {
	w, fc := getTestWatcher()

	epHandler := EpHandler{"endpoints", w.Ep}
	err := w.allNamespacesWatchFor(&epHandler, w.Cs.CoreV1().RESTClient(),
		fields.Everything(), &v1.Endpoints{}, 0, fc)

	if err != nil {
		t.Error(err)
	}

	w.Cs.CoreV1().Endpoints("trafficserver-test-2").Create(context.TODO(), &v1.Endpoints{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver-test-2",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "10.10.1.1",
					},
					{
						IP: "10.10.2.2",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "main",
						Port:     8080,
						Protocol: "TCP",
					},
				},
			},
		},
	}, meta_v1.CreateOptions{})

	time.Sleep(100 * time.Millisecond)

	w.Cs.CoreV1().Endpoints("trafficserver-test-2").Update(context.TODO(), &v1.Endpoints{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver-test-2",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "10.10.1.1",
					},
					{
						IP: "10.10.3.3",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "main",
						Port:     8080,
						Protocol: "TCP",
					},
				},
			},
		},
	}, meta_v1.UpdateOptions{})

	time.Sleep(100 * time.Millisecond)

	returnedKeys := w.Ep.RedisClient.GetDefaultDBKeyValues()
	expectedKeys := getExpectedKeysForEndpointAdd()
	expectedKeys["trafficserver-test-2:testsvc:8080"][1] = "10.10.3.3#8080#http"

	if !reflect.DeepEqual(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestAllNamespacesWatchFor_Delete(t *testing.T) {
	w, fc := getTestWatcher()

	epHandler := EpHandler{"endpoints", w.Ep}
	err := w.allNamespacesWatchFor(&epHandler, w.Cs.CoreV1().RESTClient(),
		fields.Everything(), &v1.Endpoints{}, 0, fc)

	if err != nil {
		t.Error(err)
	}

	fc.Add(&v1.Endpoints{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver-test-2",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "10.10.1.1",
					},
					{
						IP: "10.10.2.2",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "main",
						Port:     8080,
						Protocol: "TCP",
					},
				},
			},
		},
	})
	time.Sleep(100 * time.Millisecond)

	fc.Delete(&v1.Endpoints{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver-test-2",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "10.10.1.1",
					},
					{
						IP: "10.10.3.3",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "main",
						Port:     8080,
						Protocol: "TCP",
					},
				},
			},
		},
	})
	time.Sleep(100 * time.Millisecond)

	returnedKeys := w.Ep.RedisClient.GetDefaultDBKeyValues()
	expectedKeys := make(map[string][]string)

	if !reflect.DeepEqual(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestInNamespacesWatchFor_Add(t *testing.T) {
	w, _ := getTestWatcher()

	cmHandler := CMHandler{"configmaps", w.Ep}
	targetNs := make([]string, 1, 1)
	targetNs[0] = "trafficserver"

	err := w.inNamespacesWatchFor(&cmHandler, w.Cs.CoreV1().RESTClient(),
		targetNs, fields.Everything(), &v1.ConfigMap{}, 0)

	if err != nil {
		t.Error(err)
	}

	w.Cs.CoreV1().ConfigMaps("trafficserver").Create(context.TODO(), &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver",
			Annotations: map[string]string{
				"ats-configmap": "true",
			},
		},
		Data: map[string]string{
			"proxy.config.output.logfile.rolling_enabled":      "1",
			"proxy.config.output.logfile.rolling_interval_sec": "4000",
			"proxy.config.restart.active_client_threshold":     "2",
		},
	}, meta_v1.CreateOptions{})
	time.Sleep(100 * time.Millisecond)

	rEnabled, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_enabled")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rEnabled, "1") {
		t.Errorf("returned \n%s,  but expected \n%s", rEnabled, "1")
	}

	rInterval, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_interval_sec")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rInterval, "4000") {
		t.Errorf("returned \n%s,  but expected \n%s", rInterval, "4000")
	}

	threshold, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.restart.active_client_threshold")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(threshold, "2") {
		t.Errorf("returned \n%s,  but expected \n%s", threshold, "2")
	}
}

func TestInNamespacesWatchFor_Update(t *testing.T) {
	w, _ := getTestWatcher()

	cmHandler := CMHandler{"configmaps", w.Ep}
	targetNs := make([]string, 1, 1)
	targetNs[0] = "trafficserver"

	err := w.inNamespacesWatchFor(&cmHandler, w.Cs.CoreV1().RESTClient(),
		targetNs, fields.Everything(), &v1.ConfigMap{}, 0)

	if err != nil {
		t.Error(err)
	}

	w.Cs.CoreV1().ConfigMaps("trafficserver").Create(context.TODO(), &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver",
			Annotations: map[string]string{
				"ats-configmap": "true",
			},
		},
		Data: map[string]string{
			"proxy.config.output.logfile.rolling_enabled":      "1",
			"proxy.config.output.logfile.rolling_interval_sec": "4000",
			"proxy.config.restart.active_client_threshold":     "2",
		},
	}, meta_v1.CreateOptions{})
	time.Sleep(100 * time.Millisecond)

	w.Cs.CoreV1().ConfigMaps("trafficserver").Update(context.TODO(), &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver",
			Annotations: map[string]string{
				"ats-configmap": "true",
			},
		},
		Data: map[string]string{
			"proxy.config.output.logfile.rolling_enabled":      "1",
			"proxy.config.output.logfile.rolling_interval_sec": "3000",
			"proxy.config.restart.active_client_threshold":     "0",
		},
	}, meta_v1.UpdateOptions{})
	time.Sleep(100 * time.Millisecond)

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

func TestInNamespacesWatchFor_ShouldNotAdd(t *testing.T) {
	w, _ := getTestWatcher()

	cmHandler := CMHandler{"configmaps", w.Ep}
	targetNs := make([]string, 1, 1)
	targetNs[0] = "trafficserver"

	err := w.inNamespacesWatchFor(&cmHandler, w.Cs.CoreV1().RESTClient(),
		targetNs, fields.Everything(), &v1.ConfigMap{}, 0)

	if err != nil {
		t.Error(err)
	}

	w.Cs.CoreV1().ConfigMaps("trafficserver").Create(context.TODO(), &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver",
			Annotations: map[string]string{
				"ats-configmap": "true",
			},
		},
		Data: map[string]string{
			"proxy.config.output.logfile.rolling_enabled":      "1",
			"proxy.config.output.logfile.rolling_interval_sec": "4000",
			"proxy.config.restart.active_client_threshold":     "2",
		},
	}, meta_v1.CreateOptions{})
	time.Sleep(100 * time.Millisecond)

	w.Cs.CoreV1().ConfigMaps("trafficserver-2").Create(context.TODO(), &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc-2",
			Namespace: "trafficserver-2",
		},
		Data: map[string]string{
			"proxy.config.output.logfile.rolling_enabled":      "1",
			"proxy.config.output.logfile.rolling_interval_sec": "3000",
			"proxy.config.restart.active_client_threshold":     "4",
		},
	}, meta_v1.CreateOptions{})
	time.Sleep(100 * time.Millisecond)

	rEnabled, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_enabled")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rEnabled, "1") {
		t.Errorf("returned \n%s,  but expected \n%s", rEnabled, "1")
	}

	rInterval, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_interval_sec")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rInterval, "4000") {
		t.Errorf("returned \n%s,  but expected \n%s", rInterval, "4000")
	}

	threshold, err := cmHandler.Ep.ATSManager.ConfigGet("proxy.config.restart.active_client_threshold")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(threshold, "2") {
		t.Errorf("returned \n%s,  but expected \n%s", threshold, "2")
	}

	w.Cs.CoreV1().ConfigMaps("trafficserver-2").Create(context.TODO(), &v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "trafficserver",
		},
		Data: map[string]string{
			"proxy.config.output.logfile.rolling_enabled":      "1",
			"proxy.config.output.logfile.rolling_interval_sec": "3000",
			"proxy.config.restart.active_client_threshold":     "4",
		},
	}, meta_v1.CreateOptions{})
	time.Sleep(100 * time.Millisecond)

	rEnabled, err = cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_enabled")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rEnabled, "1") {
		t.Errorf("returned \n%s,  but expected \n%s", rEnabled, "1")
	}

	rInterval, err = cmHandler.Ep.ATSManager.ConfigGet("proxy.config.output.logfile.rolling_interval_sec")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(rInterval, "4000") {
		t.Errorf("returned \n%s,  but expected \n%s", rInterval, "4000")
	}

	threshold, err = cmHandler.Ep.ATSManager.ConfigGet("proxy.config.restart.active_client_threshold")

	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(threshold, "2") {
		t.Errorf("returned \n%s,  but expected \n%s", threshold, "2")
	}
}

func getTestWatcher() (Watcher, *framework.FakeControllerSource) {
	clientset := fake.NewSimpleClientset()
	fc := framework.NewFakeControllerSource()

	exampleEndpoint := createExampleEndpointWithFakeATS()
	stopChan := make(chan struct{})

	ingressWatcher := Watcher{
		Cs:           clientset,
		ATSNamespace: "trafficserver-test-2",
		Ep:           &exampleEndpoint,
		StopChan:     stopChan,
	}

	return ingressWatcher, fc
}
