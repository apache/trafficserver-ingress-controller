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

package util

import (
	"encoding/json"
	"fmt"
	"os"

	//	p "path"
	"sync"
)

// Writer writes the JSON file synchronously
type Writer struct {
	lock     sync.Mutex
	DirPath  string
	FileName string
}

// Perm is default permission bits of JSON file
const Perm os.FileMode = 0755

const (
	// Define annotations we check for in the watched resources
	AnnotationServerSnippet = "ats.ingress.kubernetes.io/server-snippet"
	AnnotationIngressClass  = "kubernetes.io/ingress.class"
)

// SyncWriteJSONFile writes obj, intended to be HostGroup, into a JSON file
// under filename.
func (w *Writer) SyncWriteJSONFile(obj interface{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	file, err := w.CreateFileIfNotExist()
	if err != nil {
		return err
	}
	defer file.Close() // file is opened, must close

	content, jsonErr := json.MarshalIndent(obj, "", "\t")
	if jsonErr != nil {
		return jsonErr
	}

	// Making sure repeated writes will actually clear the file before each write
	err = file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(content)
	if writeErr != nil {
		return writeErr
	}
	file.Sync() // flushing to disk
	return nil
}

// CreateFileIfNotExist checks if fileName in dirPath already exists.
// if not, such file will be created. If the file exists, it will be opened
// and its file descriptor returned. So, Caller needs to close the file!
func (w *Writer) CreateFileIfNotExist() (file *os.File, err error) {
	file, err = nil, nil

	if _, e := os.Stat(w.DirPath); e == nil { // dirPath exists
		// fall through
	} else if os.IsNotExist(e) { // dirPath does not exist
		err = os.MkdirAll(w.DirPath, Perm)
	} else { // sys error
		err = e
	}

	if err != nil {
		return
	}

	// caller is responsible for checking err first before using file anyways
	file, err = os.OpenFile(w.DirPath+"/"+w.FileName, os.O_CREATE|os.O_RDWR, Perm)
	return
}

// ConstructHostPathString constructs the string representation of Host + Path
func ConstructHostPathString(scheme, host, path string) string {
	if path == "" {
		path = "/"
	}
	return scheme + "://" + host + path
	//return p.Clean(fmt.Sprintf("%s/%s", host, path))
}

// ConstructSvcPortString constructs the string representation of namespace, svc, port
func ConstructSvcPortString(namespace, svc, port string) string {
	return namespace + ":" + svc + ":" + port
}

// ConstructIPPortString constructs the string representation of ip, port
func ConstructIPPortString(ip, port, protocol string) string {
	if protocol != "https" {
		protocol = "http"
	}
	return ip + "#" + port + "#" + protocol
}

func ConstructNameVersionString(namespace, name, version string) string {
	return "$" + namespace + "/" + name + "/" + version
}

// Itos : Interface to String
func Itos(obj interface{}) string {
	return fmt.Sprintf("%v", obj)
}

func ExtractServerSnippet(ann map[string]string) (snippet string, err error) {

	server_snippet, ok := ann[AnnotationServerSnippet]
	if !ok {
		return "", fmt.Errorf("missing annotation '%s'", AnnotationServerSnippet)
	}

	return server_snippet, nil
}

func ExtractIngressClass(ann map[string]string) (class string, err error) {

	ingress_class, ok := ann[AnnotationIngressClass]
	if !ok {
		return "", fmt.Errorf("missing annotation '%s'", AnnotationIngressClass)
	}

	return ingress_class, nil
}

// FmtMarshalled converts json marshalled bytes to string
func FmtMarshalled(marshalled []byte) string {
	return fmt.Sprintf("%q", marshalled)
}
