<!--
    Licensed to the Apache Software Foundation (ASF) under one
    or more contributor license agreements.  See the NOTICE file
    distributed with this work for additional information
    regarding copyright ownership.  The ASF licenses this file
    to you under the Apache License, Version 2.0 (the
    "License"); you may not use this file except in compliance
    with the License.  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing,
    software distributed under the License is distributed on an
    "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
    KIND, either express or implied.  See the License for the
    specific language governing permissions and limitations
    under the License.
-->

## Development

### Develop with Go-Lang in Linux
1. Get Go-lang 1.16.12 from [official site](https://golang.org/dl/)
2. Add `go` command to your PATH: `export PATH=$PATH:/usr/local/go/bin`
3. Define GOPATH: `export GOPATH=$(go env GOPATH)`
4. Add Go workspace to your PATH: `export PATH=$PATH:$(go env GOPATH)/bin`
5. Define the base path: `mkdir -p $GOPATH/src/github.com/`
6. Clone the project:
   - `cd $GOPATH/src/github.com/`
   - `git clone <project>`
   - `cd <project>`
7. Define GO111MODULE: `export GO111MODULE=on` to be able to compile locally. 

### Compilation
To compile, type: `go build -o ingress-ats main/main.go`

### Unit Tests
The project includes unit tests for the controller written in Golang and the ATS plugin written in Lua.

To run the Golang unit tests: `go test ./watcher/ && go test ./redis/`

The Lua unit tests use `busted` for testing. `busted` can be installed using `luarocks`:`luarocks install busted`. More information on how to install busted is available [here](https://olivinelabs.com/busted/). 
> :warning: **Note that the project uses Lua 5.1 version**

To run the Lua unit tests: 
- `cd pluginats`
- `busted connect_redis_test.lua` 

### Text-Editor
The repository comes with basic support for both [vscode](https://code.visualstudio.com/) and `vim`. 

If you're using `vscode`:
- `.vscode/settings.json` contains some basic settings for whitespaces and tabs
- `.vscode/extensions.json` contains a few recommended extensions for this project.
- It is highly recommended to install the Go extension since it contains the code lint this project used during development.

If you're using `vim`, a `vimrc` file with basic whitespace and tab configurations is also provided
