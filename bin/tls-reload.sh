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
                  
oldcksum=`cksum ${tlscrt}`
                                                                                        
inotifywait -e modify,move,create,delete -mr --timefmt '%d/%m/%y %H:%M' --format '%T' \ 
	${tlspath} | while read date time; do

		newcksum=`cksum ${tlscrt}`              
		if [ "$newcksum" != "$oldcksum" ]; then                                   
			echo "At ${time} on ${date}, tls cert/key files update detected." 
			oldcksum=$newcksum                                      
			touch /opt/ats/etc/trafficserver/ssl_multicert.config 
			traffic_ctl config reload 
                fi 
        done 

