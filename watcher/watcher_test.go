package watcher

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	dynamicfake "k8s.io/client-go/dynamic/fake"
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
					{IP: "10.10.1.1"},
					{IP: "10.10.2.2"},
				},
				Ports: []v1.EndpointPort{
					{Name: "main", Port: 8080, Protocol: "TCP"},
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
					{IP: "10.10.1.1"},
					{IP: "10.10.2.2"},
				},
				Ports: []v1.EndpointPort{
					{Name: "main", Port: 8080, Protocol: "TCP"},
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
					{IP: "10.10.1.1"},
					{IP: "10.10.3.3"},
				},
				Ports: []v1.EndpointPort{
					{Name: "main", Port: 8080, Protocol: "TCP"},
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
					{IP: "10.10.1.1"},
					{IP: "10.10.2.2"},
				},
				Ports: []v1.EndpointPort{
					{Name: "main", Port: 8080, Protocol: "TCP"},
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
					{IP: "10.10.1.1"},
					{IP: "10.10.3.3"},
				},
				Ports: []v1.EndpointPort{
					{Name: "main", Port: 8080, Protocol: "TCP"},
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
	targetNs := []string{"trafficserver"}

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
	targetNs := []string{"trafficserver"}

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
	targetNs := []string{"trafficserver"}

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
}

// getTestWatcher returns a Watcher configured with a typed fake clientset.
// It uses createExampleEndpointWithFakeATS (assumed to exist in other test code)
// and a FakeControllerSource for the informer tests.
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

// getTestWatcherForCache returns a Watcher configured with a fake dynamic client
// that knows the List kind for the ATSCachingPolicy resource.
func getTestWatcherForCache() (Watcher, *framework.FakeControllerSource) {
	scheme := runtime.NewScheme()

	gvr := schema.GroupVersionResource{
		Group:    "k8s.trafficserver.apache.com",
		Version:  "v1alpha1",
		Resource: "atscachingpolicies",
	}

	// Map the GVR to its List kind name used by the informer reflection/listing.
	gvrToListKind := map[schema.GroupVersionResource]string{
		gvr: "ATSCachingPolicyList",
	}

	// dynamic fake client
	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)

	clientset := fake.NewSimpleClientset()
	fc := framework.NewFakeControllerSource()
	exampleEndpoint := createExampleEndpointWithFakeATSCache()
	stopChan := make(chan struct{})

	ingressWatcher := Watcher{
		Cs:            clientset,
		DynamicClient: dynClient,
		ATSNamespace:  "trafficserver-test-2",
		Ep:            &exampleEndpoint,
		StopChan:      stopChan,
		ResyncPeriod:  0,
	}

	return ingressWatcher, fc
}

func filePath(t *testing.T) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "cache.config")

	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	return tmpFile
}

// --- Tests that exercise WatchAtsCachingPolicy (Add/Update/Delete) ---
// Each test starts the caching-policy watcher (which attaches AtsCacheHandler),
// then creates/updates/deletes an unstructured ATSCachingPolicy CR and finally
// calls the fake ATS manager's CacheSet() to mimic the handler's reload action.

// Test Add event triggers CacheSet
func TestWatchAtsCachingPolicy_Add(t *testing.T) {
	w, _ := getTestWatcherForCache()
	path := filePath(t)
	err := w.WatchAtsCachingPolicy(path)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "k8s.trafficserver.apache.com",
		Version:  "v1alpha1",
		Resource: "atscachingpolicies",
	}
	dynClient := w.DynamicClient.Resource(gvr).Namespace("default")

	// Create a new caching policy
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "k8s.trafficserver.apache.com/v1alpha1",
			"kind":       "ATSCachingPolicy",
			"metadata": map[string]interface{}{
				"name":      "policy-add",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{
						"pattern": "/images/*",
						"action":  "cache",
						"ttl":     "3600s",
					},
				},
			},
		},
	}

	_, err = dynClient.Create(context.TODO(), policy, meta_v1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create caching policy: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Verify CacheSet call worked
	msg, err := w.Ep.ATSManager.CacheSet()
	if err != nil {
		t.Fatalf("CacheSet failed after add: %v", err)
	}
	if msg == "" {
		t.Errorf("expected non-empty CacheSet message after add")
	}
}

// Test Update event triggers CacheSet
func TestWatchAtsCachingPolicy_Update(t *testing.T) {
	w, _ := getTestWatcherForCache()
	path := filePath(t)
	err := w.WatchAtsCachingPolicy(path)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "k8s.trafficserver.apache.com",
		Version:  "v1alpha1",
		Resource: "atscachingpolicies",
	}
	dynClient := w.DynamicClient.Resource(gvr).Namespace("default")

	// Create a policy first
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "k8s.trafficserver.apache.com/v1alpha1",
			"kind":       "ATSCachingPolicy",
			"metadata": map[string]interface{}{
				"name":      "policy-update",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{
						"pattern": "/images/*",
						"action":  "cache",
						"ttl":     "3600s",
					},
				},
			},
		},
	}

	_, err = dynClient.Create(context.TODO(), policy, meta_v1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create caching policy before update: %v", err)
	}

	// Update the policy
	policy.Object["spec"] = map[string]interface{}{
		"rules": []interface{}{
			map[string]interface{}{
				"pattern": "/videos/*",
				"action":  "cache",
				"ttl":     "7200s",
			},
		},
	}
	_, err = dynClient.Update(context.TODO(), policy, meta_v1.UpdateOptions{})
	if err != nil {
		t.Fatalf("failed to update caching policy: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Verify CacheSet call worked
	msg, err := w.Ep.ATSManager.CacheSet()
	if err != nil {
		t.Fatalf("CacheSet failed after update: %v", err)
	}
	if msg == "" {
		t.Errorf("expected non-empty CacheSet message after update")
	}
}

// Test Delete event triggers CacheSet
func TestWatchAtsCachingPolicy_Delete(t *testing.T) {
	w, _ := getTestWatcherForCache()
	path := filePath(t)
	err := w.WatchAtsCachingPolicy(path)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "k8s.trafficserver.apache.com",
		Version:  "v1alpha1",
		Resource: "atscachingpolicies",
	}
	dynClient := w.DynamicClient.Resource(gvr).Namespace("default")

	// Create a policy first
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "k8s.trafficserver.apache.com/v1alpha1",
			"kind":       "ATSCachingPolicy",
			"metadata": map[string]interface{}{
				"name":      "policy-delete",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{
						"pattern": "/docs/*",
						"action":  "cache",
						"ttl":     "1800s",
					},
				},
			},
		},
	}

	_, err = dynClient.Create(context.TODO(), policy, meta_v1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create caching policy before delete: %v", err)
	}

	// Delete the policy
	err = dynClient.Delete(context.TODO(), "policy-delete", meta_v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("failed to delete caching policy: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Verify CacheSet call worked
	msg, err := w.Ep.ATSManager.CacheSet()
	if err != nil {
		t.Fatalf("CacheSet failed after delete: %v", err)
	}
	if msg == "" {
		t.Errorf("expected non-empty CacheSet message after delete")
	}
}

func getTestWatcherForSni() Watcher {
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{
		Group:    "trafficserver.apache.org",
		Version:  "v1alpha1",
		Resource: "atssnipolicies",
	}
	gvrToListKind := map[schema.GroupVersionResource]string{
		gvr: "ATSSniPolicyList",
	}
	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
	clientset := fake.NewSimpleClientset()
	exampleEndpoint := createExampleEndpointWithFakeATSSni()
	stopChan := make(chan struct{})

	sniWatcher := Watcher{
		Cs:            clientset,
		DynamicClient: dynClient,
		ATSNamespace:  "trafficserver-test-2",
		Ep:            &exampleEndpoint,
		StopChan:      stopChan,
		ResyncPeriod:  0,
	}

	return sniWatcher
}

func tempSniFile(t *testing.T) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sni.yaml")
	if err := os.WriteFile(tmpFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	return tmpFile
}

func newSniCR(name, fqdn string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "trafficserver.apache.org/v1alpha1",
			"kind":       "ATSSniPolicy",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"sni": []interface{}{
					map[string]interface{}{
						"fqdn":            fqdn,
						"verify_client":   "STRICT",
						"host_sni_policy": "PERMISSIVE",
						"valid_tls_versions_in": []interface{}{
							"TLSv1_2",
						},
					},
				},
			},
		},
	}
}

// --- TESTS ---

func TestWatchAtsSniPolicy_Add(t *testing.T) {
	w := getTestWatcherForSni()
	path := tempSniFile(t)

	err := w.WatchAtsSniPolicy(path)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "trafficserver.apache.org",
		Version:  "v1alpha1",
		Resource: "atssnipolicies",
	}
	dynClient := w.DynamicClient.Resource(gvr).Namespace("default")

	// Create CR
	cr := newSniCR("policy-add", "ats.test.com")
	_, err = dynClient.Create(context.TODO(), cr, meta_v1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create SNI CR: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// File must contain fqdn
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "ats.test.com") {
		t.Errorf("expected fqdn ats.test.com in sni.yaml, got:\n%s", string(data))
	}
}

func TestWatchAtsSniPolicy_Update(t *testing.T) {
	w := getTestWatcherForSni()
	path := tempSniFile(t)

	err := w.WatchAtsSniPolicy(path)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "trafficserver.apache.org",
		Version:  "v1alpha1",
		Resource: "atssnipolicies",
	}
	dynClient := w.DynamicClient.Resource(gvr).Namespace("default")

	// Create CR with fqdn
	cr := newSniCR("policy-update", "ats.test.com")
	_, err = dynClient.Create(context.TODO(), cr, meta_v1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create SNI CR: %v", err)
	}

	// Update CR: keep ats.test.com, add new-site.com
	cr.Object["spec"] = map[string]interface{}{
		"sni": []interface{}{
			map[string]interface{}{
				"fqdn":            "ats.test.com",
				"verify_client":   "NONE",
				"host_sni_policy": "ENFORCE",
			},
			map[string]interface{}{
				"fqdn":            "new-site.com",
				"verify_client":   "NONE",
				"host_sni_policy": "ENFORCE",
			},
		},
	}
	_, err = dynClient.Update(context.TODO(), cr, meta_v1.UpdateOptions{})
	if err != nil {
		t.Fatalf("failed to update SNI CR: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// File should contain ats.test.com and new-site.com, but not host-test.com
	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "ats.test.com") {
		t.Errorf("expected fqdn ats.test.com in sni.yaml after update, got:\n%s", content)
	}
	if !strings.Contains(content, "new-site.com") {
		t.Errorf("expected fqdn new-site.com in sni.yaml after update, got:\n%s", content)
	}
}

func TestWatchAtsSniPolicy_Delete(t *testing.T) {
	w := getTestWatcherForSni()
	path := tempSniFile(t)

	err := w.WatchAtsSniPolicy(path)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "trafficserver.apache.org",
		Version:  "v1alpha1",
		Resource: "atssnipolicies",
	}
	dynClient := w.DynamicClient.Resource(gvr).Namespace("default")

	// Create CR with fqdn
	cr := newSniCR("policy-delete", "ats.test.com")
	_, err = dynClient.Create(context.TODO(), cr, meta_v1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create SNI CR: %v", err)
	}

	// Delete CR
	err = dynClient.Delete(context.TODO(), "policy-delete", meta_v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("failed to delete SNI CR: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// File should be empty, because both fqdn entries came from the deleted CR
	data, _ := os.ReadFile(path)
	if len(strings.TrimSpace(string(data))) != 0 {
		t.Errorf("expected sni.yaml to be empty after delete, got:\n%s", string(data))
	}
}
