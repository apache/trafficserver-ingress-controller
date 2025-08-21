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
	"fmt"
	"os/exec"
	"strings"
)

// ATSManager talks to ATS
// In the future, this is the struct that should manage
// everything related to ATS
type ATSManagerInterface interface {
	ConfigSet(k, v string) (string, error)
	ConfigGet(k string) (string, error)
	CacheSet()(string, error)
	IncludeIngressClass(c string) bool
}

type ATSManager struct {
	Namespace    string
	IngressClass string
}

func (m *ATSManager) IncludeIngressClass(c string) bool {
	if m.IngressClass == "" {
		return true
	}

	if m.IngressClass == c {
		return true
	}

	return false
}

// ConfigSet configures reloadable ATS config. When there is no error,
// a message string is returned
func (m *ATSManager) ConfigSet(k, v string) (msg string, err error) {
	cmd := exec.Command("traffic_ctl", "config", "set", k, v)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute: traffic_ctl config set %s %s Error: %s", k, v, err.Error())
	}
	return fmt.Sprintf("Ran p.Key: %s p.Val: %s --> stdoutStderr: %q", k, v, stdoutStderr), nil
}

func (m * ATSManager) CacheSet() (msg string, err error) {
	cmd := exec.Command("traffic_ctl", "config", "reload")
        stdoutStderr, err := cmd.CombinedOutput()
        if err != nil {
                return "", fmt.Errorf("failed to execute: traffic_ctl config reload  Error: %s", err.Error())
        }
        return fmt.Sprintf("Reload succesful --> stdoutStderr: %q", stdoutStderr), nil

}

func (m *ATSManager) ConfigGet(k string) (msg string, err error) {
	cmd := exec.Command("traffic_ctl", "config", "get", k)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute: traffic_ctl config get %s Error: %s", k, err.Error())
	}
	stdoutString := fmt.Sprintf("%q", stdoutStderr)
	configValue := strings.Split(strings.Trim(strings.Trim(stdoutString, "\""), "\\n"), ": ")[1]
	return configValue, err
}
