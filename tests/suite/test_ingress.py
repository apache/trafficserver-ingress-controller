#  Licensed to the Apache Software Foundation (ASF) under one
#  or more contributor license agreements.  See the NOTICE file
#  distributed with this work for additional information
#  regarding copyright ownership.  The ASF licenses this file
#  to you under the Apache License, Version 2.0 (the
#  "License"); you may not use this file except in compliance
#  with the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

import requests
import pytest
import os
import time
import textwrap
import subprocess

def kubectl_apply(yaml_path):
    os.system('kubectl apply -f ' + yaml_path)
    time.sleep(3)

def kubectl_create(resource):
    os.system('kubectl create ' + resource)
    time.sleep(1)

def kubectl_delete(resource):
    os.system('kubectl delete ' + resource)

def misc_command(command):
    rc = os.system(command)
    if rc != 0:
        # os.system returns a shell-dependent code; keep it simple:
        raise RuntimeError(f"Command failed (rc={rc}): {command}")
def create_certs():

    # Work dir
    misc_command('mkdir -p certs')

    # Root CA
    misc_command('openssl genrsa -out certs/rootCA.key 4096')
    misc_command(
        'openssl req -x509 -new -key certs/rootCA.key -sha256 -days 3650 '
        '-out certs/rootCA.crt '
        '-subj "/C=US/ST=State/L=City/O=MyOrg/OU=MyUnit/CN=TestRootCA" '
        '-addext "basicConstraints=critical,CA:TRUE" '
        '-addext "keyUsage=critical,keyCertSign,cRLSign" '
        '-addext "subjectKeyIdentifier=hash"'
    )

    #Self-Signed certificate for node-app-4
    misc_command('openssl genrsa -out ../k8s/images/node-app-4/origin.key 4096')
    misc_command(
        'openssl req -x509 -new -key ../k8s/images/node-app-4/origin.key -sha256 -days 3650 '
        '-out ../k8s/images/node-app-4/origin.crt '
        '-subj "/C=US/ST=State/L=City/O=MyOrg/OU=MyUnit/CN=test.example.com" '
    )

    # Backend CA
    misc_command('openssl genrsa -out ../k8s/images/node-app-3/backend.key 2048')
    misc_command(
        'openssl req -new -key ../k8s/images/node-app-3/backend.key '
        '-out ../k8s/images/node-app-3/backend.csr '
        '-subj "/C=US/ST=State/L=City/O=TestOrg/CN=test.example.com.backend.svc.cluster.local" '
    )
    misc_command(
        'openssl x509 -req -in ../k8s/images/node-app-3/backend.csr -CA certs/rootCA.crt -CAkey certs/rootCA.key -CAcreateserial '
        '-out ../k8s/images/node-app-3/backend.crt '
        '-days 365 -sha256 '
    )

    # Server key + CSR
    misc_command('openssl genrsa -out certs/server.key 2048')
    misc_command(
        'openssl req -new -key certs/server.key -out certs/server.csr '
        '-subj "/C=US/ST=State/L=City/O=MyOrg/OU=MyUnit/CN=test.edge.com" '
        '-addext "subjectAltName=DNS:test.edge.com"'
    )

    # Server ext file
    server_ext = textwrap.dedent("""\
        [ v3_server ]
        basicConstraints = CA:FALSE
        keyUsage = digitalSignature, keyEncipherment
        extendedKeyUsage = serverAuth
        subjectAltName = @alt_names

        [ alt_names ]
        DNS.1 = test.edge.com
    """)
    with open("certs/server_ext.cnf", "w", encoding="utf-8") as f:
        f.write(server_ext)

    # Sign server CSR
    misc_command(
        'openssl x509 -req -in certs/server.csr '
        '-CA certs/rootCA.crt -CAkey certs/rootCA.key -CAcreateserial '
        '-out certs/server.crt -days 365 -sha256 '
        '-extfile certs/server_ext.cnf -extensions v3_server'
    )
    misc_command('openssl verify -CAfile certs/rootCA.crt certs/server.crt')

    # Client key + CSR
    misc_command('openssl genrsa -out certs/client1.key 2048')
    misc_command(
        'openssl req -new -key certs/client1.key -out certs/client1.csr '
        '-subj "/C=US/ST=State/L=City/O=MyOrg/OU=QA/CN=client1"'
    )
    # Client ext file
    client_ext = textwrap.dedent("""\
        [ v3_client ]
        basicConstraints = CA:FALSE
        keyUsage = digitalSignature, keyEncipherment
        extendedKeyUsage = clientAuth
    """)
    with open("certs/client_ext.cnf", "w", encoding="utf-8") as f:
        f.write(client_ext)

    # Sign client CSR
    misc_command(
        'openssl x509 -req -in certs/client1.csr '
        '-CA certs/rootCA.crt -CAkey certs/rootCA.key -CAcreateserial '
        '-out certs/client1.crt -days 365 -sha256 '
        '-extfile certs/client_ext.cnf -extensions v3_client'
    )
    misc_command('openssl verify -CAfile certs/rootCA.crt certs/client1.crt')

    # Server2 key + CSR
    misc_command('openssl genrsa -out certs/server2.key 2048')
    misc_command(
        'openssl req -new -key certs/server2.key -out certs/server2.csr '
        '-subj "/C=US/ST=State/L=City/O=MyOrg/OU=MyUnit/CN=test.example.com" '
        '-addext "subjectAltName=DNS:test.example.com"'
    )

    # Server2 ext file
    server2_ext = textwrap.dedent("""\
        [ v3_server2 ]
        basicConstraints = CA:FALSE
        keyUsage = digitalSignature, keyEncipherment
        extendedKeyUsage = serverAuth
        subjectAltName = @alt_names

        [ alt_names ]
        DNS.1 = test.example.com
    """)
    with open("certs/server2_ext.cnf", "w", encoding="utf-8") as f:
        f.write(server2_ext)

    # Sign server CSR
    misc_command(
        'openssl x509 -req -in certs/server2.csr '
        '-CA certs/rootCA.crt -CAkey certs/rootCA.key -CAcreateserial '
        '-out certs/server2.crt -days 365 -sha256 '
        '-extfile certs/server2_ext.cnf -extensions v3_server2'
    )
    misc_command('openssl verify -CAfile certs/rootCA.crt certs/server2.crt')

def setup_module(module):
    create_certs()
    misc_command('openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=atssvc/O=atssvc"')
    kubectl_create('namespace trafficserver-test')
    kubectl_create('namespace backend')
    kubectl_create('secret tls app3-secret --key certs/server2.key --cert certs/server2.crt -n backend  --dry-run=client -o yaml | kubectl apply -f -') 
    kubectl_create('secret tls tls-secret --key tls.key --cert tls.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -')
    kubectl_create('secret tls server-secret --key certs/server.key --cert certs/server.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -')
    kubectl_create('secret tls ca-secret --key certs/rootCA.key --cert certs/rootCA.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -')
    kubectl_create('secret tls server2-secret --key certs/server2.key --cert certs/server2.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -')
    kubectl_apply('data/setup/configmaps/')
    kubectl_apply('data/setup/traffic-server/')
    kubectl_apply('data/setup/apps/')
    kubectl_apply('data/setup/ingresses/')
    kubectl_apply('../k8s/images/node-app-3/yaml/')
    kubectl_apply('../k8s/images/node-app-4/yaml/')

    #Sni crd 
    kubectl_apply('../ats_sni/ats-snipolicy-role.yaml')
    kubectl_apply('../ats_sni/ats-snipolicy-binding.yaml')
    kubectl_apply('../ats_sni/crd-atssnipolicy.yaml')
    #kubectl_apply('data/setup/ats_sni/atssnipolicy.yaml')

    #Applying here as it takes some time for controller to get notification from kubernetes.
    kubectl_apply('../ats_caching/ats-cachingpolicy-role.yaml')
    kubectl_apply('../ats_caching/ats-cachingpolicy-binding.yaml')
    kubectl_apply('../ats_caching/crd-atscachingpolicy.yaml')
    kubectl_apply('../ats_caching/atscachingpolicy.yaml')
    kubectl_apply('data/caching-app/')

    time.sleep(60)
    misc_command('kubectl get all -A')
    misc_command('kubectl get pod -A -o wide')
    misc_command('kubectl logs $(kubectl get pod -n trafficserver-test-2 -o name | head -1) -n trafficserver-test-2')
    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test-2 -o name | head -1) -n trafficserver-test-2 -- ps auxxx')
    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test-2 -o name | head -1) -n trafficserver-test-2 -- curl -v localhost:8080/app1')
    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test-2 -o name | head -1) -n trafficserver-test-2 -- curl -v $(kubectl get pod -n trafficserver-test-2 -o jsonpath={.items[0].status.podIP}):8080/app1')    
    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test-3 -o name | head -1) -n trafficserver-test-3 -- curl -v $(kubectl get pod -n trafficserver-test-2 -o jsonpath={.items[0].status.podIP}):8080/app1')    

    #    misc_command('kubectl logs $(kubectl get pod -n trafficserver-test-3 -o name | head -1) -n trafficserver-test-3')
    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- curl -v $(kubectl get pod -n trafficserver-test-2 -o jsonpath={.items[0].status.podIP}):8080/app1')
#    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- curl -v $(kubectl get pod -n trafficserver-test-3 -o jsonpath={.items[0].status.podIP}):8080/app1')
    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- curl -v $(kubectl get service/appsvc1 -n trafficserver-test-2 -o jsonpath={.spec.clusterIP}):8080/app1')
#    misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- curl -v $(kubectl get service/appsvc2 -n trafficserver-test-2 -o jsonpath={.spec.clusterIP}):8080/app1')



def teardown_module(module):
    kubectl_delete('namespace trafficserver-test-3')
    kubectl_delete('namespace trafficserver-test-2')
    kubectl_delete('namespace trafficserver-test')
    kubectl_delete('namespace cache-test-ns')
    kubectl_delete('namespace backend')
    misc_command('rm -rf certs')
    misc_command('rm ../k8s/images/node-app-3/backend.csr')
    misc_command('rm ../k8s/images/node-app-3/backend.crt')
    misc_command('rm ../k8s/images/node-app-3/backend.key')
    misc_command('rm ../k8s/images/node-app-4/origin.crt')
    misc_command('rm ../k8s/images/node-app-4/origin.key')
    
   

def get_expected_response_http2_disabled():
    resp="""<HTML>
            <HEAD>
            <TITLE>Not Found on Accelerator</TITLE>
            </HEAD>

            <BODY BGCOLOR="white" FGCOLOR="black">
            <H1>Not Found on Accelerator</H1>
            <HR>

            <FONT FACE="Helvetica,Arial"><B>
            Description: Your request on the specified host was not found.
            Check the location and try again.
            </B></FONT>
            <HR>
            </BODY>"""

    return ' '.join(resp.split())

def get_expected_response_app1():
    resp = """<!DOCTYPE html>
            <HTML>

            <HEAD>
            <TITLE>
                Hello from app1
            </TITLE>
            </HEAD>

            <BODY>
                <H1>Hi</H1>
                <P>This is very minimal "hello world" HTML document.</P>
            </BODY>
            </HTML>"""
    
    return ' '.join(resp.split())

def get_expected_response_app1_updated():
    resp = """<!DOCTYPE html>
            <HTML>

            <HEAD>
            <TITLE>
                Hello from app1 - Request to path /app2
            </TITLE>
            </HEAD>

            <BODY>
                <H1>Hi</H1>
                <P>This is very minimal "hello world" HTML document.</P>
            </BODY>
            </HTML>"""
    
    return ' '.join(resp.split())

def get_expected_response_app2():
    resp = """<!DOCTYPE html>
            <HTML>

            <HEAD>
            <TITLE>
                A Small Hello
            </TITLE>
            </HEAD>

            <BODY>
                <H1>Hi</H1>
                <P>This is very minimal "hello world" HTML document.</P>
            </BODY>
            </HTML>"""
    
    return ' '.join(resp.split())


class TestIngress:
    def test_basic_routing_edge_app1(self, minikubeip):
        req_url = "http://" + minikubeip + ":30080/app1"
        resp = requests.get(req_url, headers={"host": "test.edge.com"})
        time.sleep(5)
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1()
        
    def test_basic_routing_media_app1(self, minikubeip):
        req_url = "http://" + minikubeip + ":30080/app1"
        resp = requests.get(req_url, headers={"host": "test.media.com"})
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1()
    
    def test_basic_routing_edge_app2(self, minikubeip):
        req_url = "http://" + minikubeip + ":30080/app2"
        resp = requests.get(req_url, headers={"host": "test.edge.com"})
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app2()
    
    def test_basic_routing_media_app2(self, minikubeip):
        req_url = "http://" + minikubeip + ":30080/app2"
        resp = requests.get(req_url, headers={"host": "test.media.com"})
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app2()
    
    def test_basic_routing_edge_app2_https(self, minikubeip):
        req_url = "https://" + minikubeip + ":30443/app2"
        resp = requests.get(req_url, headers={"host": "test.edge.com"}, verify=False)
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app2()

    def test_cache_app1(self, minikubeip):
        command = f'curl -i -v -H "Host: test.media.com" http://{minikubeip}:30080/cache-test'
        response_1 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response1 = response_1.stdout.strip()
        response1_list = response1.split('\n')
        for res in response1_list:
            if res.__contains__("Age"):
                age1 = res
            if res.__contains__("Date"):
                mod_time1 = res
        time.sleep(5)
        response_2 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response2 = response_2.stdout.strip()
        response2_list = response2.split('\n')
        for resp in response2_list:
            if resp.__contains__("Age"):
                age2 = resp
            if resp.__contains__("Date"):
                mod_time2 = resp
        assert mod_time1 == mod_time2 and age1 != age2, "Expected Date provided by both responses to be same and the Age mentioned in second response to be more than 0"

    def test_cache_https_node_app3(self, minikubeip):
        command = f'curl -i -v --cacert certs/rootCA.crt --resolve test.example.com:30443:{minikubeip} https://test.example.com:30443/node-app3'
        response_1 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response1 = response_1.stdout.strip()
        response1_list = response1.split('\n')
        for res in response1_list:
            if res.__contains__("age"):
                age1 = res
            if res.__contains__("date"):
                mod_time1 = res
        time.sleep(5)
        response_2 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response2 = response_2.stdout.strip()
        response2_list = response2.split('\n')
        for resp in response2_list:
            if resp.__contains__("age"):
                age2 = resp
            if resp.__contains__("date"):
                mod_time2 = resp
        assert mod_time1 == mod_time2 and age1 != age2, "Expected Date provided by both responses to be same and the Age mentioned in second response to be more than 0"


    def test_cache_app1_beyond_ttl(self, minikubeip):
        # waiting for cache from previous test case to expire
        time.sleep(20)

        command = f'curl -i -v -H "Host: test.media.com" http://{minikubeip}:30080/cache-test'
        response_1 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response1 = response_1.stdout.strip()
        response1_list = response1.split('\n')
        for res in response1_list:
            if res.__contains__("Age"):
                age1 = res
            if res.__contains__("Date"):
                mod_time1 = res
        time.sleep(16)
        response_2 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response2 = response_2.stdout.strip()
        response2_list = response2.split('\n')
        for resp in response2_list:
            if resp.__contains__("Age"):
                age2 = resp
            if resp.__contains__("Date"):
                mod_time2 = resp       
        expected_age = "Age: 0"
        assert mod_time1 != mod_time2 and age1 == age2 and age2 == expected_age, "Expected Date provided by both responses should be different and the Age mentioned in both responses should be 0"

    def test_cache_app2(self, minikubeip):
        command = f'curl -i -v -H "Host: test.edge.com" http://{minikubeip}:30080/app2'
        response_1 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response1 = response_1.stdout.strip()
        response1_list = response1.split('\n')
        for res in response1_list:
            if res.__contains__("Age"):
                age1 = res
            if res.__contains__("Date"):
                mod_time1 = res
        time.sleep(9)
        response_2 = subprocess.run(command, shell=True, capture_output=True, text=True)
        response2 = response_2.stdout.strip()
        response2_list = response2.split('\n')
        for resp in response2_list:
            if resp.__contains__("Age"):
                age2 = resp
            if resp.__contains__("Date"):
                mod_time2 = resp
        kubectl_delete('crd atscachingpolicies.k8s.trafficserver.apache.com')
        assert mod_time1 != mod_time2 and age1 == age2, "Expected Date provided by both the responses to be different and the Age to be 0 in both the responses"
    
    def test_updating_ingress_media_app2(self, minikubeip):
        kubectl_apply('data/ats-ingress-update.yaml')
        req_url = "http://" + minikubeip + ":30080/app2"
        resp = requests.get(req_url, headers={"host": "test.media.com"})
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1_updated()
    
    def test_deleting_ingress_media_app2(self, minikubeip):
        kubectl_apply('data/ats-ingress-delete.yaml')
        req_url = "http://" + minikubeip + ":30080/app2"
        resp = requests.get(req_url, headers={"host": "test.media.com"})
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 404,\
            f"Expected: 400 response code for test_basic_routing_deleted_ingress"

    def test_add_ingress_media(self, minikubeip):
        kubectl_apply('data/ats-ingress-add.yaml')
        req_url = "http://" + minikubeip + ":30080/test"
        resp = requests.get(req_url, headers={"host": "test.media.com"})
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1()

    def test_snippet_edge_app2(self, minikubeip):
        kubectl_apply('data/ats-ingress-snippet.yaml')
        req_url = "http://" + minikubeip + ":30080/app2"
        resp = requests.get(req_url, headers={"host": "test.edge.com"},allow_redirects=False)
        misc_command('kubectl exec $(kubectl get pod -n trafficserver-test -o name) -n trafficserver-test -- cat /opt/ats/var/log/trafficserver/squid.log')

        assert resp.status_code == 301,\
            f"Expected: 301 response code for test_snippet_edge_app2"
        assert resp.headers['Location'] == 'https://test.edge.com/app2'
    
    def test_https2_enabled(self, minikubeip):
        kubectl_apply('../ats_sni/http2/on.yaml')
        time.sleep(10)  # wait for config changes propagate
    
        cmd = f'curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl HTTPS/2 request failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/2 200" in result.stderr, "Expected HTTP/2 200 response"
        expected = get_expected_response_app2()
        actual = ' '.join(result.stdout.split())  # normalize whitespace
        assert expected == actual, "Response body did not match expected"

    def test_https2_disabled(self, minikubeip):
        kubectl_apply('../ats_sni/http2/off.yaml')
        time.sleep(5)
        cmd = f'curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl HTTPS/2 request failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1" in result.stderr, "Expected HTTP/1.1"
        expected = get_expected_response_http2_disabled()
        actual = ' '.join(result.stdout.split())
        assert expected == actual, "Response body did not match expected"

    def test_verify_client_none(self, minikubeip):
        kubectl_apply('../ats_sni/verify-client/none.yaml')
        time.sleep(7)  
        cmd = f'curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in result.stderr or "HTTP/2 200" in result.stderr

    def test_verify_client_moderate_without_crt(self, minikubeip):
        kubectl_apply('../ats_sni/verify-client/moderate.yaml')
        time.sleep(7)
        cmd = f'curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in result.stderr or "HTTP/2 200" in result.stderr

    def test_verify_client_moderate_with_crt(self, minikubeip):
        kubectl_apply('../ats_sni/verify-client/moderate.yaml')
        time.sleep(7)
        cmd = f'curl --cacert certs/rootCA.crt --cert certs/client1.crt --key certs/client1.key -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in result.stderr or "HTTP/2 200" in result.stderr
        
    def test_verify_client_strict_with_crt(self, minikubeip):
        kubectl_apply('../ats_sni/verify-client/strict.yaml')
        time.sleep(7)
        cmd = f'curl --cacert certs/rootCA.crt --cert certs/client1.crt --key certs/client1.key -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in result.stderr or "HTTP/2 200" in result.stderr

    def test_verify_client_strict_without_crt(self, minikubeip):
        
        kubectl_apply('../ats_sni/verify-client/strict.yaml')
        time.sleep(7)
        cmd = f'curl --cacert certs/rootCA.crt  -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode != 0, "Curl unexpectedly succeeded without client certificate"
        expected_error = "tlsv13 alert certificate required"
        assert expected_error in result.stderr, (
        f"Expected TLS failure not found. stderr:\n{result.stderr}"
        )

    def test_host_sni_none(self, minikubeip):
        kubectl_apply('../ats_sni/host-sni-policy/disabled.yaml')
        time.sleep(7)
        cmd = f'curl --cacert certs/rootCA.crt  -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in result.stderr or "HTTP/2 200" in result.stderr

    def test_host_sni_match_enforced(self, minikubeip):
        kubectl_apply('../ats_sni/host-sni-policy/enforced.yaml')
        time.sleep(7)
        cmd = f'curl --cacert certs/rootCA.crt  -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in result.stderr or "HTTP/2 200" in result.stderr

    def test_host_sni_mismatch_enforced(self, minikubeip):
        time.sleep(7)
        cmd = (
            f'curl -v --cacert certs/rootCA.crt '
            f'--resolve test.example.com:30443:{minikubeip} '
            f'https://test.example.com:30443/node-app3 '
            f'-H "Host: test.edge.com"'
        )
        # Execute curl
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        # Validate TLS handshake and HTTP response
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/2 403" in result.stderr or "HTTP/1.1 403" in result.stderr, "Expected 403 Access Denied"
        log_cmd = (
            "kubectl exec $(kubectl get pod -n trafficserver-test -o name | head -1) "
            "-n trafficserver-test -- "
            "grep -i 'SNI/hostname mismatch sni=test.example.com host=test.edge.com action=terminate' "
            "/opt/ats/var/log/trafficserver/diags.log | sed 's/.*\\(SNI\\/hostname mismatch.*\\)/\\1/'"
        )
        log_result = subprocess.run(log_cmd, shell=True, capture_output=True, text=True)
        assert "SNI/hostname mismatch sni=test.example.com host=test.edge.com action=terminate" in log_result.stdout, (
            f"Expected log entry not found. Logs:\n{log_result.stdout}"
        )

    def test_host_sni_match_permissive(self, minikubeip):
        kubectl_apply('../ats_sni/host-sni-policy/permissive.yaml')
        time.sleep(7)
        cmd = f'curl --cacert certs/rootCA.crt  -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in result.stderr or "HTTP/2 200" in result.stderr   

    def test_host_sni_mismatch_permissive(self, minikubeip):
        time.sleep(7)
        cmd = (
            f'curl -v --cacert certs/rootCA.crt '
            f'--resolve test.example.com:30443:{minikubeip} '
            f'https://test.example.com:30443/node-app3 '
            f'-H "Host: test.edge.com"'
        )
        # Execute curl
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        # Validate TLS handshake and HTTP response
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in result.stderr, "TLS handshake failed"
        assert "HTTP/2 404" in result.stderr or "HTTP/1.1 404" in result.stderr
        log_cmd = (
            "kubectl exec $(kubectl get pod -n trafficserver-test -o name | head -1) "
            "-n trafficserver-test -- "
            "grep -i 'SNI/hostname mismatch sni=test.example.com host=test.edge.com action=continue' "
            "/opt/ats/var/log/trafficserver/diags.log | sed 's/.*\\(SNI\\/hostname mismatch.*\\)/\\1/'"
        )
        log_result = subprocess.run(log_cmd, shell=True, capture_output=True, text=True)
        assert "SNI/hostname mismatch sni=test.example.com host=test.edge.com action=continue" in log_result.stdout, (
            f"Expected log entry not found. Logs:\n{log_result.stdout}"
        )
    
    # ==================== ENFORCED MODE ====================

    def test_verify_server_enforced_with_valid_cert(self, minikubeip):
        """Test ENFORCED mode with valid certificate (backend.crt) - should succeed with 200 OK"""
        kubectl_apply('../ats_sni/verify-server-policy/enforced.yaml')
        time.sleep(7)

        cmd = f'curl -v --cacert certs/rootCA.crt --resolve test.example.com:30443:{minikubeip} https://test.example.com:30443/node-app3'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        full_output = result.stdout + result.stderr

        # Should succeed with 200 OK (valid cert passes strict verification)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in full_output, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in full_output or "HTTP/2 200" in full_output or "200 OK" in full_output, \
            f"Expected 200 OK. Got: {result.stdout[:200]}"

        print("ENFORCED mode with valid cert: 200 OK")

    def test_verify_server_enforced_with_invalid_cert(self, minikubeip):
        """Test ENFORCED mode with invalid backend certificate (origin.crt) - should fail with 502"""
        kubectl_apply('../ats_sni/verify-server-policy/enforced.yaml')
        time.sleep(7)

        cmd = f'curl -v --cacert certs/rootCA.crt --resolve test.example.com:30443:{minikubeip} https://test.example.com:30443/node-app4'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        full_output = result.stdout + result.stderr

        # Should get 502 error (ENFORCED = rejects invalid cert)
        assert result.returncode == 0, f"Curl command itself failed: {result.stderr}"

        # Check for 502 error
        has_502 = "HTTP/2 502" in full_output or "Could Not Connect" in full_output or "502 Bad Gateway" in full_output
        assert has_502, f"Expected 502 error in ENFORCED mode with invalid cert. Got: {full_output[:500]}"

        print("Got 502 error, now checking logs for Action=Terminate...")

        # Wait for logs
        time.sleep(10)

        # Get pod name
        get_pod_cmd = "kubectl get pods -n trafficserver-test -l app=trafficserver-test -o jsonpath='{.items[0].metadata.name}'"
        pod_result = subprocess.run(get_pod_cmd, shell=True, capture_output=True, text=True)
        pod_name = pod_result.stdout.strip().replace("'", "")

        # Check logs for Action=Terminate
        log_cmd = f"kubectl exec -n trafficserver-test {pod_name} -- grep 'Action=Terminate' /opt/ats/var/log/trafficserver/diags.log | tail -5"
        log_result = subprocess.run(log_cmd, shell=True, capture_output=True, text=True)

        if log_result.stdout:
            print(f"Termination logs found:\n{log_result.stdout}")
            assert "Action=Terminate" in log_result.stdout, "Expected Action=Terminate"

        print("ENFORCED mode with invalid cert: 502 error (connection terminated)")

    def test_verify_server_disabled_with_valid_cert(self, minikubeip):
        """Test DISABLED mode with valid certificate (backend.crt) - should succeed with 200 OK"""
        kubectl_apply('../ats_sni/verify-server-policy/disabled.yaml')
        time.sleep(7)
        
        cmd = f'curl -v --cacert certs/rootCA.crt --resolve test.example.com:30443:{minikubeip} https://test.example.com:30443/node-app3'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        full_output = result.stdout + result.stderr
        
        # Should succeed with 200 OK (DISABLED = no verification)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in full_output, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in full_output or "HTTP/2 200" in full_output or "200 OK" in full_output, \
            f"Expected 200 OK. Got: {result.stdout[:200]}"
        
        print("DISABLED mode with valid cert: 200 OK")

    def test_verify_server_disabled_with_invalid_cert(self, minikubeip):
        """Test DISABLED mode with invalid backend certificate (origin.crt) - should succeed with 200 OK"""
        kubectl_apply('../ats_sni/verify-server-policy/disabled.yaml')
        time.sleep(7)
        
        cmd = f'curl -v --cacert certs/rootCA.crt --resolve test.example.com:30443:{minikubeip} https://test.example.com:30443/node-app4'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        full_output = result.stdout + result.stderr
        
        # Should ALSO succeed with 200 OK (DISABLED = no verification, accepts any cert)
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in full_output, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in full_output or "HTTP/2 200" in full_output or "200 OK" in full_output, \
            f"Expected 200 OK. Got: {result.stdout[:200]}"
        
        print("DISABLED mode with invalid cert: 200 OK (no verification performed)")


    # ==================== PERMISSIVE MODE ====================
    def test_verify_server_permissive_with_valid_cert(self, minikubeip):
        """Test PERMISSIVE mode with valid certificate (backend.crt) - should succeed with 200 OK"""
        kubectl_apply('../ats_sni/verify-server-policy/permissive.yaml')
        time.sleep(7)
        
        cmd = f'curl -v --cacert certs/rootCA.crt --resolve test.example.com:30443:{minikubeip} https://test.example.com:30443/node-app3'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        full_output = result.stdout + result.stderr
        
        # Should succeed with 200 OK
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in full_output, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in full_output or "HTTP/2 200" in full_output or "200 OK" in full_output, \
            f"Expected 200 OK. Got: {result.stdout[:200]}"
        
        print("PERMISSIVE mode with valid cert: 200 OK")


    def test_verify_server_permissive_with_invalid_cert(self, minikubeip):
        """Test PERMISSIVE mode with invalid backend certificate (origin.crt) - should succeed with 200 OK and log warnings"""
        kubectl_apply('../ats_sni/verify-server-policy/permissive.yaml')
        time.sleep(7)
        
        # Connect to Flask on 8449 with self-signed origin.crt
        cmd = f'curl -v --cacert certs/rootCA.crt --resolve test.example.com:30443:{minikubeip} https://test.example.com:30443/node-app4'
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        full_output = result.stdout + result.stderr
        
        misc_command('kubectl get pods -n backend')
        misc_command('kubectl get pods -n trafficserver-test-2')
        
        misc_command('kubectl get all -A')

        misc_command('kubectl describe pods -n backend')

        misc_command('kubectl get pods -n backend -o name | xargs -n1 kubectl logs --prefix -n backend')
        
        
        assert result.returncode == 0, f"Curl failed: {result.stderr}"
        assert "SSL connection using TLS" in full_output, "TLS handshake failed"
        assert "HTTP/1.1 200 OK" in full_output or "HTTP/2 200" in full_output or "200 OK" in full_output, \
            f"Expected 200 OK. Got: {result.stdout[:200]}"
        
        print("Got 200 OK, now checking logs for warnings...")
        
        # Wait for logs to be written
        time.sleep(10)
        
        # Get pod name
        get_pod_cmd = "kubectl get pods -n trafficserver-test -l app=trafficserver-test -o jsonpath='{.items[0].metadata.name}'"
        pod_result = subprocess.run(get_pod_cmd, shell=True, capture_output=True, text=True)
        pod_name = pod_result.stdout.strip().replace("'", "")
        
        assert pod_name, f"TrafficServer pod not found"
        
        # Search for warnings in logs
        log_cmd = f"kubectl exec -n trafficserver-test {pod_name} -- grep -i 'Core server certificate verification failed' /opt/ats/var/log/trafficserver/diags.log | tail -100"
        log_result = subprocess.run(log_cmd, shell=True, capture_output=True, text=True)
        
        # Verify warnings are present
        if not log_result.stdout:
            recent_cmd = f"kubectl exec -n trafficserver-test {pod_name} -- tail -200 /opt/ats/var/log/trafficserver/diags.log"
            recent_result = subprocess.run(recent_cmd, shell=True, capture_output=True, text=True)
            pytest.fail(f"No warnings found in logs. Recent logs:\n{recent_result.stdout[-500:]}")
        
        print(f"Certificate warnings found:\n{log_result.stdout}")
        
        assert "Core server certificate verification failed" in log_result.stdout, \
            "Expected certificate verification warning"
        assert "Action=Continue" in log_result.stdout, \
            "Expected 'Action=Continue' in PERMISSIVE mode"
        
        print("PERMISSIVE mode with invalid cert: 200 OK with warnings logged")
        kubectl_delete('crd atssnipolicies.trafficserver.apache.org')
