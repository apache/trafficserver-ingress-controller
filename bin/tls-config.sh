#!/usr/bin/env bash
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set +x

if [ -z "${POD_TLS_PATH}" ]; then
	echo "POD_TLS_PATH not defined"
	exit 1
fi

tlspath="$POD_TLS_PATH/"      
tlskey="$POD_TLS_PATH/tls.key"
tlscrt="$POD_TLS_PATH/tls.crt"
        
if [ ! -f "${tlscrt}" ]; then
	echo "${tlscrt} not found"
	exit 1
fi

if [ ! -f "${tlskey}" ]; then
	echo "${tlskey} not found"
	exit 1
fi

echo "dest_ip=* ssl_cert_name=${tlscrt} ssl_key_name=${tlskey}" > /usr/local/etc/trafficserver/ssl_multicert.config
