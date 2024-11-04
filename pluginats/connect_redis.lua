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

ts.add_package_cpath('/opt/ats/lib/lua/5.1/?.so;/opt/ats/lib/lua/5.1/socket/?.so;/opt/ats/lib/lua/5.1/mime/?.so')
ts.add_package_path('/opt/ats/share/lua/5.1/?.lua;/opt/ats/share/lua/5.1/socket/?.lua')

local redis = require 'redis'

-- connecting to unix domain socket
local client = redis.connect('unix:///opt/ats/var/run/redis/redis.sock')

local snippet_enabled = false

function __init__(argtb)
  if (#argtb) > 0 then
    ts.debug("Parameter is given. Snippet is enabled.")
    snippet_enabled = true
  end
end

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

function check_path_exact_match(req_scheme, req_host, req_path)
  local host_path = "E+"..req_scheme .. "://" .. req_host .. req_path
  ts.debug('checking host_path: '..host_path)
  client:select(1) -- go with hostpath table first
  return client:smembers(host_path) -- redis blocking call
end

function check_path_prefix_match(req_scheme, req_host, req_path)
  local host_path = "P+"..req_scheme .. "://" .. req_host .. req_path
  ts.debug('checking host_path: '..host_path)
  client:select(1)
  local svcs = client:smembers(host_path) -- redis blocking call

  if (svcs ~= nil and #svcs > 0) then
    return svcs
  end

  -- finding location of / in request path
  local t = {}                   -- table to store the indices
  local i = 0
  while true do
    i = string.find(req_path, "%/", i+1)    -- find 'next' dir
    if i == nil then break end
    table.insert(t, i)
  end

  for index = #t, 1, -1 do
    local pathindex = t[index]
    local subpath = string.sub (req_path, 1, pathindex)

    host_path = "P+"..req_scheme .. "://" .. req_host .. subpath
    ts.debug('checking host_path: '..host_path)
    client:select(1)
    svcs =client:smembers(host_path) -- redis blocking call
    if (svcs ~= nil and #svcs > 0) then
      return svcs
    end

    if pathindex > 1 then
      subpath = string.sub (req_path, 1, pathindex - 1)

      host_path = "P+"..req_scheme .. "://" .. req_host .. subpath
      ts.debug('checking host_path: '..host_path)
      client:select(1)
      svcs = client:smembers(host_path) -- redis blocking call
      if (svcs ~= nil and #svcs > 0) then
        return svcs
      end
    end
  end

  return nil
end

function get_wildcard_domain(req_host)
  local pos = string.find( req_host, '%.' )
  if pos == nil then
    return nil
  end
  return "*" .. string.sub (req_host, pos)
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

  local wildcard_req_host = get_wildcard_domain(req_host)
  ts.debug("-----Request-----")
  ts.debug("req_scheme: "..req_scheme)
  ts.debug("req_host: " .. req_host)
  ts.debug("req_port: " .. req_port)
  ts.debug("req_path: " .. req_path)
  ts.debug("wildcard_req_host: " .. (wildcard_req_host or 'invalid domain name'))
  ts.debug("-----------------")

  -- check for path exact match
  local svcs = check_path_exact_match(req_scheme, req_host, req_path)

  if (svcs == nil or #svcs == 0) then
    -- check for path prefix match
    svcs = check_path_prefix_match(req_scheme, req_host, req_path)
  end

  if (svcs == nil or #svcs == 0) and wildcard_req_host ~= nil then
    -- check for path exact match with wildcard domain name in prefix
    svcs = check_path_exact_match(req_scheme, wildcard_req_host, req_path)
  end

  if (svcs == nil or #svcs == 0) and wildcard_req_host ~= nil then
    -- check for path prefix match with wildcard domain name in prefix
    svcs = check_path_prefix_match(req_scheme, wildcard_req_host, req_path)
  end

  if (svcs == nil or #svcs == 0) then
    -- check for path exact match with wildcard domain name
    svcs = check_path_exact_match(req_scheme, '*', req_path)
  end

  if (svcs == nil or #svcs == 0) then
    -- check for path prefix match with wildcard domain name
    svcs = check_path_prefix_match(req_scheme, '*', req_path)
  end

  if (svcs == nil or #svcs == 0) then
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

  if snippet_enabled == true then
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
end

