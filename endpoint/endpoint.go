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

package endpoint

import (
	//	"fmt"
	//	"log"

	//	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	//	v1beta1 "k8s.io/api/extensions/v1beta1"
	//	extensionsV1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"

	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"ingress-ats/namespace"
	"ingress-ats/proxy"
	"ingress-ats/redis"
	//	t "ingress-ats/types"
	//	"ingress-ats/util"
)

const (
	// UpdateRedis for readability
	UpdateRedis bool = true
	// UpdateATS for readability
	UpdateATS bool = true
)

// Endpoint stores all essential information to act on HostGroups
type Endpoint struct {
	RedisClient *redis.Client
	ATSManager  proxy.ATSManagerInterface
	NsManager   *namespace.NsManager
}
