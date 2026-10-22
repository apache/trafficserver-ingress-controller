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

// Validation harness for the HIGH/critical findings in the 2026-08-11 security
// scan (scan dir: scan-trafficserver-ingress-controller-20260811).
//
// Each test asserts the SECURE invariant the corresponding candidate patch is
// meant to establish. Against the current (unpatched) code these tests FAIL,
// and the failure output is the runtime proof that the finding is real. Once
// the candidate patch (PATCHES/bug_0N) is applied, the matching test passes —
// so this file doubles as the before/after regression check.
//
// Covered here (miniredis + FakeATSManager, no cluster needed):
//
//	f003 -> TestValidation_F003_*   (cross-tenant route injection on add)
//	f004 -> TestValidation_F004_*   (silent route deletion/takeover on update)
//	f002 -> TestValidation_F002_*   (unallowlisted ConfigMap key -> live ATS)
//
// Out of scope for a Go unit test (validated statically / other harness):
//
//	f001 (Lua loadstring sandbox)  -> needs a Lua/busted harness; none present.
//	f008 (ClusterRole secrets grant) -> RBAC YAML; see the shell assertion in
//	     the validation report.
package watcher

import (
	"testing"

	ep "github.com/apache/trafficserver-ingress-controller/endpoint"
	"github.com/apache/trafficserver-ingress-controller/namespace"
	"github.com/apache/trafficserver-ingress-controller/proxy"
	"github.com/apache/trafficserver-ingress-controller/redis"

	v1 "k8s.io/api/core/v1"
	nv1 "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newTenantEndpoint builds an Endpoint wired to a fresh in-memory Redis and a
// FakeATSManager with the shipped defaults (empty ingress class => match all,
// empty namespace maps => watch all), mirroring the stock deployment posture
// the scan assumes.
func newTenantEndpoint(t *testing.T) *ep.Endpoint {
	t.Helper()

	rClient, err := redis.InitForTesting()
	if err != nil {
		t.Fatalf("redis.InitForTesting: %v", err)
	}

	nsManager := namespace.NsManager{
		NamespaceMap:       make(map[string]bool),
		IgnoreNamespaceMap: make(map[string]bool),
	}
	nsManager.Init()

	return &ep.Endpoint{
		RedisClient: rClient,
		ATSManager: &proxy.FakeATSManager{
			Namespace:    "trafficserver",
			IngressClass: "",
			Config:       make(map[string]string),
		},
		NsManager: &nsManager,
	}
}

// singleHostIngress builds a minimal Ingress: one namespace, one host, one exact
// path, one service backend. resourceVersion lets update() see a real change.
func singleHostIngress(name, ns, host, path, svc string, port int32, resourceVersion string) nv1.Ingress {
	pt := nv1.PathTypeExact
	return nv1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:            name,
			Namespace:       ns,
			ResourceVersion: resourceVersion,
		},
		Spec: nv1.IngressSpec{
			Rules: []nv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: nv1.IngressRuleValue{
						HTTP: &nv1.HTTPIngressRuleValue{
							Paths: []nv1.HTTPIngressPath{
								{
									Path:     path,
									PathType: &pt,
									Backend: nv1.IngressBackend{
										Service: &nv1.IngressServiceBackend{
											Name: svc,
											Port: nv1.ServiceBackendPort{Number: port},
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
}

// setContains reports whether want is a member of the route-key set.
func setContains(set []string, want string) bool {
	for _, m := range set {
		if m == want {
			return true
		}
	}
	return false
}

// --------------------------------------------------------------------------
// f003 — No host ownership check lets any tenant hijack any hostname's traffic
// handlerIngress.go:122  add() -> DBOneSAdd(hostPath, svcport), no owner check
// Candidate fix: PATCHES/bug_03 (first-writer-wins ownership index)
// --------------------------------------------------------------------------
func TestValidation_F003_CrossTenantRouteInjection(t *testing.T) {
	e := newTenantEndpoint(t)
	ig := IgHandler{ResourceName: "ingresses", Ep: e}

	const host = "shop.example.com"
	const routeKey = "E+http://shop.example.com/"

	// Tenant "team-victim" legitimately claims shop.example.com first.
	victim := singleHostIngress("victim-ing", "team-victim", host, "/", "victim-backend", 8080, "1")
	ig.add(&victim)

	// Tenant "team-attacker", in a DIFFERENT namespace, claims the SAME host.
	attacker := singleHostIngress("attacker-ing", "team-attacker", host, "/", "attacker-backend", 8080, "1")
	ig.add(&attacker)

	route := e.RedisClient.GetDBOneKeyValues()[routeKey]

	// SECURE invariant (first-writer-wins): once team-victim owns the host, a
	// claim from another namespace must be refused — the attacker's backend
	// must NOT appear in the victim's route set.
	if setContains(route, "team-attacker:attacker-backend:8080") {
		t.Errorf("f003 CONFIRMED: attacker backend injected into another tenant's route %q = %v; "+
			"traffic to %s can now be split to team-attacker (candidate fix: bug_03)",
			routeKey, route, host)
	}
	if !setContains(route, "team-victim:victim-backend:8080") {
		t.Errorf("f003: victim backend missing from %q = %v (unexpected)", routeKey, route)
	}
}

// --------------------------------------------------------------------------
// f004 — Ingress update overwrites shared host/path Redis key, deleting victim
// handlerIngress.go:318/329-332  new-spec temp key is unseeded, then
// SUNIONSTORE replaces the live set. Candidate fix: PATCHES/bug_04
// (seed new-side temp keys from live).
// --------------------------------------------------------------------------
func TestValidation_F004_SilentRouteDeletionOnUpdate(t *testing.T) {
	e := newTenantEndpoint(t)
	ig := IgHandler{ResourceName: "ingresses", Ep: e}

	const host = "shop.example.com"
	const routeKey = "E+http://shop.example.com/"

	// team-victim owns shop.example.com/.
	victim := singleHostIngress("victim-ing", "team-victim", host, "/", "victim-backend", 8080, "1")
	ig.add(&victim)

	// team-attacker starts on its OWN host, then UPDATES its Ingress to point
	// at the victim's host. Because shop.example.com is only in the new spec
	// (not the old), its temp key is never seeded from the live set, and the
	// finalizing SUNIONSTORE replaces the live route with attacker-only.
	attackerOld := singleHostIngress("attacker-ing", "team-attacker", "attacker-own.example.com", "/", "attacker-backend", 8080, "1")
	ig.add(&attackerOld)
	attackerNew := singleHostIngress("attacker-ing", "team-attacker", host, "/", "attacker-backend", 8080, "2")
	ig.update(&attackerOld, &attackerNew)

	route := e.RedisClient.GetDBOneKeyValues()[routeKey]

	// SECURE invariant: an update by another tenant must not delete the
	// victim's backend from a route the victim owns.
	if !setContains(route, "team-victim:victim-backend:8080") {
		t.Errorf("f004 CONFIRMED: victim backend silently deleted by another tenant's update; "+
			"%q = %v (candidate fix: bug_04)", routeKey, route)
	}
}

// --------------------------------------------------------------------------
// f002 — Annotated ConfigMap sets arbitrary live ATS config via traffic_ctl
// handlerConfigmap.go:53  every Data key applied via ConfigSet, no allowlist
// Candidate fix: PATCHES/bug_02 (record allowlist + CONFIGMAP_RECORD_ALLOWLIST)
// --------------------------------------------------------------------------
func TestValidation_F002_UnallowlistedConfigMapKeyApplied(t *testing.T) {
	e := newTenantEndpoint(t)
	cm := CMHandler{"configmap", e}

	// A ConfigMap in the ATS namespace carrying the trusted annotation, but
	// with security-relevant records that no allowlist should permit.
	dangerous := map[string]string{
		"proxy.config.ssl.client.verify.server.policy": "DISABLED", // upstream TLS verification off
		"proxy.config.http.push_method_enabled":        "1",        // cache injection via PUSH
	}
	configMap := v1.ConfigMap{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        "tuning",
			Namespace:   "trafficserver",
			Annotations: map[string]string{"ats-configmap": "true"},
		},
		Data: dangerous,
	}
	cm.Add(&configMap)

	// SECURE invariant: dangerous, non-allowlisted records must NOT reach the
	// live ATS configuration.
	for k := range dangerous {
		if v, err := cm.Ep.ATSManager.ConfigGet(k); err == nil {
			t.Errorf("f002 CONFIRMED: unallowlisted record %q=%q pushed to live ATS "+
				"(candidate fix: bug_02)", k, v)
		}
	}
}
