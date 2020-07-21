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

package proxy

import (
	"errors"
	"fmt"
)

type FakeATSManager struct {
	Namespace    string
	IngressClass string
	Config       map[string]string
}

func (m *FakeATSManager) IncludeIngressClass(c string) bool {
	if m.IngressClass == "" {
		return true
	}

	if m.IngressClass == c {
		return true
	}

	return false
}

func (m *FakeATSManager) ConfigSet(k, v string) (msg string, err error) {
	m.Config[k] = v
	return fmt.Sprintf("Ran p.Key: %s p.Val: %s", k, v), nil
}

func (m *FakeATSManager) ConfigGet(k string) (msg string, err error) {
	if val, ok := m.Config[k]; ok {
		return val, nil
	}
	return "", errors.New("key does not exist")
}
