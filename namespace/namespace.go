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

package namespace

// import v1 "k8s.io/api/core/v1"

// ALL Namespaces constant for unified text
const ALL string = "all"

// NsManager is a function wrapper that is used to determine
// if a namespace n should be included
type NsManager struct {
	NamespaceMap       map[string]bool
	IgnoreNamespaceMap map[string]bool
	allNamespaces      bool
}

// IncludeNamespace is exported method to determine if a
// namespace should be included
func (m *NsManager) IncludeNamespace(n string) bool {
	if m.allNamespaces {
		_, prs := m.IgnoreNamespaceMap[n]
		return !prs
	}
	_, ns_prs := m.NamespaceMap[n]
	_, ins_prs := m.IgnoreNamespaceMap[n]
	if ns_prs && !ins_prs {
		return ns_prs
	}
	return false
}

// Init initializes NsManager's Namespaces
func (m *NsManager) Init() {
	if len(m.NamespaceMap) == 0 {
		m.allNamespaces = true
	} else {
		m.allNamespaces = false
	}
}

func (m *NsManager) DisableAllNamespaces() {
	m.allNamespaces = false
}
