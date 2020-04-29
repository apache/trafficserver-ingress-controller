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

ts.add_package_cpath('/usr/local/lib/lua/5.1/socket/?.so;/usr/local/lib/lua/5.1/mime/?.so')
ts.add_package_path('/usr/local/share/lua/5.1/?.lua;/usr/local/share/lua/5.1/socket/?.lua')

local redis = require 'redis'

-- connecting to unix domain socket
local client = redis.connect('unix:///var/run/redis/redis.sock')

-- helper function to split a string
function ipport_split(s, delimiter)
  result = {}
  if (s ~= nil and s ~= '') then
    for match in (s..delimiter):gmatch("(.-)"..delimiter) do
      table.insert(result, match)
    end
  end
  return result
end

---------------------------------------------
----------------- DO_REMAP ------------------
---------------------------------------------
function do_global_read_request()
  ts.debug("In do_global_read_request()==========")
  local response = client:ping()

  -- if cannot connect to redis client, terminate early
  if not response then 
    ts.debug("In 'not response: '", response)
    return 0
  end

  -- We only care about host, path, and port#
  local req_scheme = ts.client_request.get_url_scheme() or 'http'
  local req_host = ts.client_request.get_url_host() or ''
  local req_path = ts.client_request.get_uri() or ''
  local req_port = ts.client_request.get_url_port() or ''
  ts.debug("-----Request-----")
  ts.debug("req_scheme: "..req_scheme)
  ts.debug("req_host: " .. req_host)
  ts.debug("req_port: " .. req_port)
  ts.debug("req_path: " .. req_path)
  ts.debug("-----------------")

  local host_path = req_scheme .. "://" .. req_host .. req_path
  
  client:select(1) -- go with hostpath table first
  local svcs = client:smembers(host_path) -- redis blocking call
  -- host/path not in redis DB
  if svcs == nil then
    ts.error("Redis Lookup Failure: svcs == nil for hostpath")
    return 0 
  end

  for _, svc in ipairs(svcs) do
    if svc == nil then
      ts.error("Redis Lookup Failure: svc == nil for hostpath")
      return 0
    end 
    if string.sub(svc, 1, 1) ~= "$" then
      ts.debug("routing")
      client:select(0) -- go with svc table second
      local ipport = client:srandmember(svc) -- redis blocking call
      -- svc not in redis DB
      if ipport == nil then
        ts.error("Redis Lookup Failure: ipport == nil for svc")
        return 0
      end

      -- find protocol, ip , port info
      local values = ipport_split(ipport, '#');
      if #values ~= 3 then
        ts.error("Redis Lookup Failure: wrong format - "..ipport)
        return 0
      end

      ts.http.skip_remapping_set(1)
      ts.client_request.set_url_scheme(values[3])
      ts.client_request.set_uri(req_path)
      ts.client_request.set_url_host(values[1])
      ts.client_request.set_url_port(values[2])
    end 
  end

  for _, svc in ipairs(svcs) do               
    if svc == nil then
      ts.error("Redis Lookup Failure: svc == nil for hostpath")        
      return 0                                                       
    end                                                                     
    if string.sub(svc, 1, 1) == "$" then                               
      ts.debug("snippet")                    
      client:select(1)           
      local snippets = client:smembers(svc)        
                                
      if snippets == nil then
        ts.error("Redis Lookup Failure: snippets == nil for hostpath")
        return 0      
      end                                    
            
      local snippet = snippets[1]
      if snippet == nil then
        ts.error("Redis Lookup Failure: snippet == nil for hostpath")
        return 0                                                 
      end                                                
                                                    
      local f = loadstring(snippet)                      
      f()                      
    end                               
  end                               

end
