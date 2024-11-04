--  Licensed to the Apache Software Foundation (ASF) under one
--  or more contributor license agreements.  See the NOTICE file
--  distributed with this work for additional information
--  regarding copyright ownership.  The ASF licenses this file
--  to you under the Apache License, Version 2.0 (the
--  "License"); you may not use this file except in compliance
--  with the License.  You may obtain a copy of the License at
--
--  http://www.apache.org/licenses/LICENSE-2.0
--
--  Unless required by applicable law or agreed to in writing, software
--  distributed under the License is distributed on an "AS IS" BASIS,
--  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
--  See the License for the specific language governing permissions and
--  limitations under the License.

_G.ts = { client_request = {}, http = {} }
_G.client = {dbone = {}, dbdefault = {}, selecteddb = 0}
_G.TS_LUA_REMAP_DID_REMAP = 1

function ts.client_request.get_url_scheme()
    return 'http'
end

function ts.client_request.get_url_host()
    return 'test.edge.com'
end

function ts.client_request.get_uri()
    return '/app1'
end
function ts.client_request.get_url_port()
    return '80'
end

function connect(path)
  return client
end

function client.select(self, number)
  if number == 1 then
    self.selecteddb = 1
  elseif number == 0 then
    self.selecteddb = 0
  end
end

function client.sadd(self, key, ...)
  db = nil
  if self.selecteddb == 1 then 
    db = self.dbone
  elseif self.selecteddb == 0 then
    db = self.dbdefault
  end
  
  if type(db[key]) ~= "table" then
    db[key] = {}
  end

  for i=1,select('#',...) do   
    local tmp = select(i,...)
    table.insert(db[key],tmp)
  end
end

function client.ping()
  return "PONG"
end

function client.smembers(self, key)
  db = nil
  if self.selecteddb == 1 then 
    db = self.dbone
  elseif self.selecteddb == 0 then
    db = self.dbdefault
  end

  return db[key]
end

function client.srandmember(self, key)
  idx = math.random(1,2)
  db = nil
  if self.selecteddb == 1 then 
    db = self.dbone
  elseif self.selecteddb == 0 then
    db = self.dbdefault
  end
  
  return db[key][idx]
end
  

describe("Unit tests - Lua", function()
  describe("Ingress Controller", function()

    setup(function()
      local match = require("luassert.match")

      package.loaded.redis = nil
      local redis = {}
      redis.connect = {}
      redis.connect = connect
      package.preload['redis'] = function () 
        return redis
      end

      client = redis.connect()

      client:select(1)
      client:sadd("E+http://test.edge.com/app1","trafficserver-test-2:appsvc1:8080")
      client:select(0)
      client:sadd("trafficserver-test-2:appsvc1:8080","172.17.0.3#8080#http","172.17.0.5#8080#http")
      --require 'pl.pretty'.dump(client)

      stub(ts, "add_package_cpath")
      stub(ts, "add_package_path")
      stub(ts, "debug")
      stub(ts, "error")
      stub(ts.client_request, "set_url_host")
      stub(ts.client_request, "set_url_port")
      stub(ts.client_request, "set_url_scheme")
      stub(ts.client_request, "set_uri")
      stub(ts.http, "skip_remapping_set")
      stub(ts.http, "set_resp")
    end)

    it("Test - Redirect to correct IP", function()
      require("connect_redis")
      local result = do_global_read_request()

      assert.stub(ts.client_request.set_url_host).was.called_with(match.is_any_of(match.is_same("172.17.0.3"),match.is_same("172.17.0.5"))) 
      assert.stub(ts.client_request.set_url_port).was.called_with("8080")
      assert.stub(ts.client_request.set_url_scheme).was.called_with("http")
    end)

    it("Test - Snippet", function()
      client:select(1)
      client:sadd("E+http://test.edge.com/app1","$trafficserver-test-3/app-ingress/411990")
      snippet = "ts.debug('Debug msg example')\nts.error('Error msg example')\n-- ts.hook(TS_LUA_HOOK_SEND_RESPONSE_HDR, function()\n--   ts.client_response.header['Location'] = 'https://test.edge.com/app2'\n-- end)\nts.http.skip_remapping_set(0)\nts.http.set_resp(301, 'Redirect')\nts.debug('Uncomment the above lines to redirect http request to https')\nts.debug('Modification for testing')\n"
      client:sadd("$trafficserver-test-3/app-ingress/411990",snippet) 
      snippet_enabled = true
                        
      --require 'pl.pretty'.dump(client)
      require "connect_redis"
      local result = do_global_read_request()

      assert.stub(ts.error).was.called_with("Error msg example")
      assert.stub(ts.http.skip_remapping_set).was.called_with(0)
      assert.stub(ts.http.set_resp).was.called_with(301,"Redirect")
    end)

  end)
end)
