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
    os.system(command)

def setup_module(module):
    misc_command('openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=atssvc/O=atssvc"')
    misc_command('openssl genrsa -out ca.key 4096 ')
    misc_command('openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "/CN=TestCA" ')
    misc_command('openssl req -new -newkey rsa:2048 -nodes -keyout edge.key -out edge.csr -subj "/CN=test.edge.com" -addext "subjectAltName=DNS:test.edge.com"')
    misc_command('openssl x509 -req -in edge.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out edge.crt -days 365 -sha256 -extfile <(printf "subjectAltName=DNS:test.edge.com") ')

    kubectl_create('namespace trafficserver-test')
    kubectl_create('secret tls tls-secret --key tls.key --cert tls.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -')
    kubectl_create('secret tls host-secret --key edge.key --cert edge.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -')
    kubectl_apply('data/setup/configmaps/')
    kubectl_apply('data/setup/traffic-server/')
    kubectl_apply('data/setup/apps/')
    kubectl_apply('data/setup/ingresses/')

    #Applying here as it takes some time for controller to get notification from kubernetes.
    kubectl_apply('data/setup/ats_caching/ats-cachingpolicy-role.yaml')
    kubectl_apply('data/setup/ats_caching/ats-cachingpolicy-binding.yaml')
    kubectl_apply('data/setup/ats_caching/crd-atscachingpolicy.yaml')
    kubectl_apply('data/setup/ats_caching/atscachingpolicy.yaml')
    kubectl_apply('data/caching-app/')

    time.sleep(90)
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
    #kubectl_delete('namespace trafficserver-test-3')
    #kubectl_delete('namespace trafficserver-test-2')
    #kubectl_delete('namespace trafficserver-test')
    #kubectl_delete('namespace cache-test-ns')
    pass
    

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

    def test_cache_app1_beyond_ttl(self, minikubeip):
        # waiting for cache from previous test case to expire
        time.sleep(13)

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
        # waiting for cache from previous test case to expire
        time.sleep(13)
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
        kubectl_delete('-f data/setup/ats_caching/ats-cachingpolicy-role.yaml')
        kubectl_delete('-f data/setup/ats_caching/ats-cachingpolicy-binding.yaml')
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


