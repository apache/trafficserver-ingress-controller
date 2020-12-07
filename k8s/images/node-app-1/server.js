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

// Constants
const PORT = 8080;
const HOST = '0.0.0.0';

// App
const app = express();
app.get('/', (req, res) => {
  res.send('Hello world\n');
});

app.get('/test', (req, res) => { // lgtm[js/missing-rate-limiting]
  res.sendFile('hello.html', {root: __dirname });
})

app.get('/app1', (req, res) => { // lgtm[js/missing-rate-limiting]
  res.sendFile('hello.html', {root: __dirname });
})

app.get('/app2', (req, res) => { // lgtm[js/missing-rate-limiting]
  res.sendFile('hello-updated.html', {root: __dirname });
})


app.listen(PORT, HOST);
console.log(`Running on http://${HOST}:${PORT}`);
