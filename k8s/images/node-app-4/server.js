// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

'use strict';

const express = require('express');
const https = require('https');
const fs = require('fs');
const path = require('path');

// Constants
const PORT = 8443;
const HOST = '0.0.0.0';

// Read TLS cert/key mounted from k8s secret at /etc/ats/ssl/server2
const options = {
  cert: fs.readFileSync('origin.crt'),
  key: fs.readFileSync('origin.key'),
};

const app = express();

// serve static HTML file(s) from project dir (assumes hello.html exists)
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'hello.html'));
});

app.get('/test', (req, res) => {
  res.sendFile(path.join(__dirname, 'hello.html'));
});

app.get('/node-app4', (req, res) => {
  res.sendFile(path.join(__dirname, 'hello.html'));
});

// simple healthcheck used by k8s probes
app.get('/health', (req, res) => res.status(200).send('OK'));

https.createServer(options, app).listen(PORT, HOST, () => {
  console.log(`HTTPS server running on https://${HOST}:${PORT}`);
});

