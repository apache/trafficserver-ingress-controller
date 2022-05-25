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
	"testing"

	ep "github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/namespace"
	"github.com/apache/trafficserver-ingress-controller/proxy"
	"github.com/apache/trafficserver-ingress-controller/redis"
	"github.com/apache/trafficserver-ingress-controller/util"

	nv1 "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var pathExact nv1.PathType = nv1.PathTypeExact

func TestAdd_ExampleIngress(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngress()

	igHandler.add(&exampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()

	expectedKeys := getExpectedKeysForAdd()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestAdd_ExampleIngressWithAnnotation(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngressWithAnnotation()

	igHandler.add(&exampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()

	expectedKeys := getExpectedKeysForAddWithAnnotation()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestAdd_ExampleIngressWithTLS(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngressWithTLS()

	igHandler.add(&exampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()

	expectedKeys := getExpectedKeysForAdd()
	expectedKeys["E+https://test.edge.com/app1"] = expectedKeys["E+http://test.edge.com/app1"]
	delete(expectedKeys, "E+http://test.edge.com/app1")

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestAdd_ExampleIngressWithIgnoredNamespace(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngressWithTLS()

	igHandler.Ep.NsManager.IgnoreNamespaceMap["ignored-namespace"] = true

	exampleIngress.ObjectMeta.Namespace = "ignored-namespace"

	igHandler.add(&exampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()

	expectedKeys := make(map[string][]string)

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestAdd_ExampleIngressWithIncludedNamespace(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngress()

	igHandler.Ep.NsManager.DisableAllNamespaces()
	igHandler.Ep.NsManager.NamespaceMap["trafficserver-test"] = true

	igHandler.add(&exampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()

	expectedKeys := getExpectedKeysForAdd()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}

}

func TestUpdate_ModifyIngress(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngress()
	updatedExampleIngress := createExampleIngress()

	updatedExampleIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[1].Path = "/app2-modified"
	updatedExampleIngress.Spec.Rules[1].IngressRuleValue.HTTP.Paths[0].Backend.Service.Name = "appsvc1-modified"
	updatedExampleIngress.Spec.Rules[1].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port.Number = 9090

	igHandler.add(&exampleIngress)
	igHandler.update(&exampleIngress, &updatedExampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()

	expectedKeys := getExpectedKeysForUpdate_ModifyIngress()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_DeletePath(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngress()
	updatedExampleIngress := createExampleIngress()

	updatedExampleIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths = updatedExampleIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[:1]

	igHandler.add(&exampleIngress)
	igHandler.update(&exampleIngress, &updatedExampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForUpdate_DeleteService()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_ModifySnippet(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngressWithAnnotation()
	updatedExampleIngress := createExampleIngressWithAnnotation()

	exampleSnippet := getExampleSnippet()
	exampleSnippet = exampleSnippet + `
	ts.debug('Modifications for the purpose of testing')`

	updatedExampleIngress.ObjectMeta.Annotations["ats.ingress.kubernetes.io/server-snippet"] = exampleSnippet
	updatedExampleIngress.SetResourceVersion("10")

	igHandler.add(&exampleIngress)
	igHandler.update(&exampleIngress, &updatedExampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForUpdate_ModifySnippet()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_ModifyTLS(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngress()
	updatedExampleIngress := createExampleIngressWithTLS()

	igHandler.add(&exampleIngress)
	igHandler.update(&exampleIngress, &updatedExampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	expectedKeys["E+https://test.edge.com/app1"] = expectedKeys["E+http://test.edge.com/app1"]
	expectedKeys["E+http://test.edge.com/app1"] = []string{}

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDelete(t *testing.T) {
	igHandler := createExampleIgHandler()
	exampleIngress := createExampleIngress()

	igHandler.delete(&exampleIngress)

	returnedKeys := igHandler.Ep.RedisClient.GetDBOneKeyValues()

	expectedKeys := make(map[string][]string)

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}

}

func createExampleIngressWithTLS() nv1.Ingress {
	exampleIngress := createExampleIngress()

	exampleIngress.Spec.TLS = []nv1.IngressTLS{
		{
			Hosts: []string{"test.edge.com"},
		},
	}

	return exampleIngress
}

func createExampleIngressWithAnnotation() nv1.Ingress {
	exampleIngress := createExampleIngress()

	exampleIngress.ObjectMeta.Annotations = make(map[string]string)
	exampleIngress.ObjectMeta.Annotations["ats.ingress.kubernetes.io/server-snippet"] = getExampleSnippet()
	exampleIngress.Spec.Rules = exampleIngress.Spec.Rules[1:]

	return exampleIngress
}

func createExampleIngress() nv1.Ingress {
	exampleIngress := nv1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "example-ingress",
			Namespace: "trafficserver-test",
		},
		Spec: nv1.IngressSpec{
			Rules: []nv1.IngressRule{
				{
					Host: "test.media.com",
					IngressRuleValue: nv1.IngressRuleValue{
						HTTP: &nv1.HTTPIngressRuleValue{
							Paths: []nv1.HTTPIngressPath{
								{
									Path:     "/app1",
									PathType: &pathExact,
									Backend: nv1.IngressBackend{
										Service: &nv1.IngressServiceBackend{
											Name: "appsvc1",
											Port: nv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
								{
									Path:     "/app2",
									PathType: &pathExact,
									Backend: nv1.IngressBackend{
										Service: &nv1.IngressServiceBackend{
											Name: "appsvc2",
											Port: nv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Host: "test.edge.com",
					IngressRuleValue: nv1.IngressRuleValue{
						HTTP: &nv1.HTTPIngressRuleValue{
							Paths: []nv1.HTTPIngressPath{
								{
									Path:     "/app1",
									PathType: &pathExact,
									Backend: nv1.IngressBackend{
										Service: &nv1.IngressServiceBackend{
											Name: "appsvc1",
											Port: nv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return exampleIngress
}

func createExampleIgHandler() IgHandler {
	exampleEndpoint := createExampleEndpoint()
	igHandler := IgHandler{"ingresses", &exampleEndpoint}

	return igHandler
}

func createExampleEndpoint() ep.Endpoint {
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
		ATSManager: &proxy.ATSManager{
			Namespace:    "default",
			IngressClass: "",
		},
		NsManager: &nsManager,
	}

	return exampleEndpoint
}

func getExpectedKeysForUpdate_ModifySnippet() map[string][]string {
	expectedKeys := getExpectedKeysForAddWithAnnotation()

	updatedSnippet := getExampleSnippet()
	updatedSnippet = updatedSnippet + `
	ts.debug('Modifications for the purpose of testing')`

	expectedKeys["$trafficserver-test/example-ingress/10"] = []string{}
	expectedKeys["$trafficserver-test/example-ingress/10"] = append(expectedKeys["$trafficserver-test/example-ingress/10"], updatedSnippet)

	expectedKeys["E+http://test.edge.com/app1"] = expectedKeys["E+http://test.edge.com/app1"][:1]
	expectedKeys["E+http://test.edge.com/app1"] = append(expectedKeys["E+http://test.edge.com/app1"], "$trafficserver-test/example-ingress/10")

	return expectedKeys
}

func getExpectedKeysForUpdate_ModifyIngress() map[string][]string {
	expectedKeys := getExpectedKeysForAdd()

	expectedKeys["E+http://test.media.com/app2"] = []string{}

	expectedKeys["E+http://test.media.com/app2-modified"] = []string{}
	expectedKeys["E+http://test.media.com/app2-modified"] = append(expectedKeys["E+http://test.media.com/app2"], "trafficserver-test:appsvc2:8080")

	expectedKeys["E+http://test.edge.com/app1"] = []string{}
	expectedKeys["E+http://test.edge.com/app1"] = append(expectedKeys["E+http://test.edge.com/app1"], "trafficserver-test:appsvc1-modified:9090")

	return expectedKeys
}

func getExpectedKeysForUpdate_DeleteService() map[string][]string {
	expectedKeys := getExpectedKeysForAdd()

	expectedKeys["E+http://test.media.com/app2"] = []string{}

	return expectedKeys
}

func getExpectedKeysForAdd() map[string][]string {
	expectedKeys := make(map[string][]string)
	expectedKeys["E+http://test.edge.com/app1"] = []string{}
	expectedKeys["E+http://test.media.com/app1"] = []string{}
	expectedKeys["E+http://test.media.com/app2"] = []string{}

	expectedKeys["E+http://test.edge.com/app1"] = append(expectedKeys["E+http://test.edge.com/app1"], "trafficserver-test:appsvc1:8080")
	expectedKeys["E+http://test.media.com/app2"] = append(expectedKeys["E+http://test.media.com/app2"], "trafficserver-test:appsvc2:8080")
	expectedKeys["E+http://test.media.com/app1"] = append(expectedKeys["E+http://test.media.com/app1"], "trafficserver-test:appsvc1:8080")

	return expectedKeys
}

func getExpectedKeysForAddWithAnnotation() map[string][]string {
	expectedKeys := getExpectedKeysForAdd()

	delete(expectedKeys, "E+http://test.media.com/app1")
	delete(expectedKeys, "E+http://test.media.com/app2")

	expectedKeys["E+http://test.edge.com/app1"] = append(expectedKeys["E+http://test.edge.com/app1"], "$trafficserver-test/example-ingress/")

	exampleSnippet := getExampleSnippet()

	expectedKeys["$trafficserver-test/example-ingress/"] = []string{}
	expectedKeys["$trafficserver-test/example-ingress/"] = append(expectedKeys["$trafficserver-test/example-ingress/"], exampleSnippet)

	return expectedKeys
}

func getExampleSnippet() string {
	return `ts.debug('Debug msg example')
	ts.error('Error msg example')
	-- ts.hook(TS_LUA_HOOK_SEND_RESPONSE_HDR, function()
	--   ts.client_response.header['Location'] = 'https://test.edge.com/app2'
	-- end)
	-- ts.http.skip_remapping_set(0)
	-- ts.http.set_resp(301, 'Redirect')
	ts.debug('Uncomment the above lines to redirect http request to https')`
}
