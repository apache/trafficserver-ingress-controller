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
*/package watcher

import (
	"ingress-ats/util"
	"testing"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdd_BasicEndpoint(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := getExpectedKeysForEndpointAdd()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestAdd_IgnoreEndpointNamespace(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.Ep.NsManager.IgnoreNamespaceMap["trafficserver-test-2"] = true

	epHandler.add(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := make(map[string][]string)

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_UpdateAddress(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)

	exampleV1Endpoint.Subsets[0].Addresses[0].IP = "10.10.3.3"

	epHandler.update(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := getExpectedKeysForAddressUpdate()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_UpdatePortNumber(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)

	exampleV1Endpoint.Subsets[0].Ports[0].Port = 8081

	epHandler.update(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := getExpectedKeysForEndpointAdd()
	expectedKeys["trafficserver-test-2:testsvc:8081"] = make([]string, 2)
	expectedKeys["trafficserver-test-2:testsvc:8081"][0] = "10.10.2.2#8081#http"
	expectedKeys["trafficserver-test-2:testsvc:8081"][1] = "10.10.1.1#8081#http"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_UpdatePortName(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)

	exampleV1Endpoint.Subsets[0].Ports[0].Name = "https"

	epHandler.update(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := getExpectedKeysForEndpointAdd()
	expectedKeys["trafficserver-test-2:testsvc:8080"] = make([]string, 2)
	expectedKeys["trafficserver-test-2:testsvc:8080"][0] = "10.10.1.1#8080#https"
	expectedKeys["trafficserver-test-2:testsvc:8080"][1] = "10.10.2.2#8080#https"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_UpdateEndpointName(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)

	exampleV1Endpoint.ObjectMeta.Name = "testsvc-modified"

	epHandler.update(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := getExpectedKeysForEndpointAdd()
	expectedKeys["trafficserver-test-2:testsvc-modified:8080"] = expectedKeys["trafficserver-test-2:testsvc:8080"]

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDelete_DeleteEndpoint(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)
	epHandler.delete(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := make(map[string][]string)

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_DeleteAddress(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)

	exampleV1Endpoint.Subsets[0].Addresses = exampleV1Endpoint.Subsets[0].Addresses[:1]

	epHandler.update(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := getExpectedKeysForEndpointAdd()
	expectedKeys["trafficserver-test-2:testsvc:8080"] = expectedKeys["trafficserver-test-2:testsvc:8080"][:1]

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestUpdate_AddAddress(t *testing.T) {
	epHandler := createExampleEpHandler()
	exampleV1Endpoint := createExampleV1Endpoint()

	epHandler.add(&exampleV1Endpoint)

	exampleV1Endpoint.Subsets[0].Addresses = append(exampleV1Endpoint.Subsets[0].Addresses, v1.EndpointAddress{
		IP: "10.10.3.3",
	})

	epHandler.update(&exampleV1Endpoint)

	returnedKeys := epHandler.Ep.RedisClient.GetDefaultDBKeyValues()

	expectedKeys := getExpectedKeysForEndpointAdd()
	expectedKeys["trafficserver-test-2:testsvc:8080"] = append(expectedKeys["trafficserver-test-2:testsvc:8080"], "10.10.3.3#8080#http")

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func createExampleV1Endpoint() v1.Endpoints {
	exampleEndpoint := v1.Endpoints{
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
	}

	return exampleEndpoint
}

func createExampleEpHandler() EpHandler {
	exampleEndpoint := createExampleEndpoint()
	epHandler := EpHandler{"endpoints", &exampleEndpoint}

	return epHandler
}

func getExpectedKeysForEndpointAdd() map[string][]string {
	expectedKeys := make(map[string][]string)
	expectedKeys["trafficserver-test-2:testsvc:8080"] = []string{}

	expectedKeys["trafficserver-test-2:testsvc:8080"] = append(expectedKeys["trafficserver-test-2:testsvc:8080"], "10.10.1.1#8080#http")
	expectedKeys["trafficserver-test-2:testsvc:8080"] = append(expectedKeys["trafficserver-test-2:testsvc:8080"], "10.10.2.2#8080#http")

	return expectedKeys
}

func getExpectedKeysForAddressUpdate() map[string][]string {
	expectedKeys := getExpectedKeysForEndpointAdd()

	if expectedKeys["trafficserver-test-2:testsvc:8080"][0] == "10.10.1.1#8080#http" {
		expectedKeys["trafficserver-test-2:testsvc:8080"] = expectedKeys["trafficserver-test-2:testsvc:8080"][1:]
	} else {
		expectedKeys["trafficserver-test-2:testsvc:8080"] = expectedKeys["trafficserver-test-2:testsvc:8080"][:1]
	}

	expectedKeys["trafficserver-test-2:testsvc:8080"] = append(expectedKeys["trafficserver-test-2:testsvc:8080"], "10.10.3.3#8080#http")

	return expectedKeys
}
