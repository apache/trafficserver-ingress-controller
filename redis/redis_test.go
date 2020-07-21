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
package redis

import (
	"ingress-ats/util"
	"testing"
)

func TestInit(t *testing.T) {
	_, err := InitForTesting()
	if err != nil {
		t.Error(err)
	}
}

func TestFlush(t *testing.T) {
	rClient, _ := InitForTesting()
	rClient.DefaultDB.SAdd("test-key", "test-val")
	rClient.DefaultDB.SAdd("test-key", "test-val-2")

	err := rClient.Flush()
	if err != nil {
		t.Error(err)
	}

	returnedKeys := rClient.GetDefaultDBKeyValues()
	expectedKeys := make(map[string][]string)
	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestGetDefaultDBKeyValues(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DefaultDB.SAdd("test-key", "test-val")
	rClient.DefaultDB.SAdd("test-key", "test-val-2")
	rClient.DefaultDB.SAdd("test-key-2", "test-val")

	returnedKeys := rClient.GetDefaultDBKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	expectedKeys["test-key"] = append([]string{"test-val-2"}, expectedKeys["test-key"]...)
	expectedKeys["test-key-2"] = make([]string, 1)
	expectedKeys["test-key-2"][0] = "test-val"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestGetDBOneKeyValues(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DBOne.SAdd("test-key", "test-val")
	rClient.DBOne.SAdd("test-key", "test-val-2")
	rClient.DBOne.SAdd("test-key-2", "test-val")

	returnedKeys := rClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	expectedKeys["test-key"] = append([]string{"test-val-2"}, expectedKeys["test-key"]...)
	expectedKeys["test-key-2"] = make([]string, 1)
	expectedKeys["test-key-2"][0] = "test-val"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDefaultDBSAdd(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DefaultDBSAdd("test-key", "test-val")
	returnedKeys := rClient.GetDefaultDBKeyValues()
	expectedKeys := getExpectedKeysForAdd()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDefaultDBDel(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DefaultDBSAdd("test-key", "test-val")
	rClient.DefaultDBSAdd("test-key-2", "test-val-2")
	rClient.DefaultDBDel("test-key")

	returnedKeys := rClient.GetDefaultDBKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	delete(expectedKeys, "test-key")
	expectedKeys["test-key-2"] = make([]string, 1)
	expectedKeys["test-key-2"][0] = "test-val-2"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDefaultDBSUnionStore(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DefaultDBSAdd("test-key", "test-val")
	rClient.DefaultDBSAdd("test-key-2", "test-val-2")
	rClient.DefaultDBSUnionStore("test-key", "test-key-2")

	returnedKeys := rClient.GetDefaultDBKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	expectedKeys["test-key"][0] = "test-val-2"
	expectedKeys["test-key-2"] = make([]string, 1)
	expectedKeys["test-key-2"][0] = "test-val-2"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDBOneSAdd(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DBOneSAdd("test-key", "test-val")
	returnedKeys := rClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForAdd()

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDBOneSRem(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DBOneSAdd("test-key", "test-val")
	rClient.DBOneSAdd("test-key", "test-val-2")
	rClient.DBOneSAdd("test-key", "test-val-3")
	rClient.DBOneSRem("test-key", "test-val-2")
	returnedKeys := rClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	expectedKeys["test-key"] = append([]string{"test-val-3"}, expectedKeys["test-key"]...)

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDBOneDel(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DBOneSAdd("test-key", "test-val")
	rClient.DBOneSAdd("test-key-2", "test-val-2")
	rClient.DBOneDel("test-key")

	returnedKeys := rClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	delete(expectedKeys, "test-key")
	expectedKeys["test-key-2"] = make([]string, 1)
	expectedKeys["test-key-2"][0] = "test-val-2"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func TestDBOneSUnionStore(t *testing.T) {
	rClient, _ := InitForTesting()

	rClient.DBOneSAdd("test-key", "test-val")
	rClient.DBOneSAdd("test-key-2", "test-val-2")
	rClient.DBOneSUnionStore("test-key", "test-key-2")

	returnedKeys := rClient.GetDBOneKeyValues()
	expectedKeys := getExpectedKeysForAdd()
	expectedKeys["test-key"][0] = "test-val-2"
	expectedKeys["test-key-2"] = make([]string, 1)
	expectedKeys["test-key-2"][0] = "test-val-2"

	if !util.IsSameMap(returnedKeys, expectedKeys) {
		t.Errorf("returned \n%v,  but expected \n%v", returnedKeys, expectedKeys)
	}
}

func getExpectedKeysForAdd() map[string][]string {
	expectedKeys := make(map[string][]string)
	expectedKeys["test-key"] = make([]string, 1)
	expectedKeys["test-key"][0] = "test-val"
	return expectedKeys
}
