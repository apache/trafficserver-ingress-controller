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

# TLS auto reload script
/usr/local/bin/tls-reload.sh >> /usr/local/var/log/ingress/ingress_ats.err &

# generate TLS cert config file for ats 
/usr/local/bin/tls-config.sh 

# append specific environment variables to records.config 
/usr/local/bin/records-config.sh

# append extra plugins to plugin.config
if [ ! -f "${EXTRA_PLUGIN_FNAME}" ]; then
	cat $EXTRA_PLUGIN_FNAME >> /usr/local/etc/trafficserver/plugin.config
fi

# start redis
redis-server /usr/local/etc/redis.conf 

# create health check file and start ats
touch /var/run/ts-alive
chown -R nobody:nobody /usr/local/etc/trafficserver
DISTRIB_ID=gentoo /usr/local/bin/trafficserver start

if [ -z "${INGRESS_NS}" ]; then
	INGRESS_NS="all"
fi

sleep 20 
/usr/local/go/bin/src/ingress-ats/ingress_ats -atsIngressClass="$INGRESS_CLASS" -atsNamespace="$POD_NAMESPACE" -namespaces="$INGRESS_NS" -ignoreNamespaces="$INGRESS_IGNORE_NS" -useInClusterConfig=T 2>>/usr/local/var/log/ingress/ingress_ats.err

