import requests
import pytest
import os
import time
import textwrap

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
    kubectl_create('namespace trafficserver-test')
    kubectl_create('secret tls tls-secret --key tls.key --cert tls.crt -n trafficserver-test --dry-run=client -o yaml | kubectl apply -f -')
    kubectl_apply('../k8s/configmaps/')
    kubectl_apply('../k8s/traffic-server/')
    kubectl_apply('../k8s/apps/')
    kubectl_apply('../k8s/ingresses/')
    time.sleep(90)

def teardown_module(module):
    kubectl_delete('namespace trafficserver-test-3')
    kubectl_delete('namespace trafficserver-test-2')
    kubectl_delete('namespace trafficserver-test')

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
        req_url = "http://" + minikubeip + ":30000/app1"
        resp = requests.get(req_url, headers={"host": "test.edge.com"})

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1()
        
    def test_basic_routing_media_app1(self, minikubeip):
        req_url = "http://" + minikubeip + ":30000/app1"
        resp = requests.get(req_url, headers={"host": "test.media.com"})

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1()
    
    def test_basic_routing_edge_app2(self, minikubeip):
        req_url = "http://" + minikubeip + ":30000/app2"
        resp = requests.get(req_url, headers={"host": "test.edge.com"})

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app2()
    
    def test_basic_routing_media_app2(self, minikubeip):
        req_url = "http://" + minikubeip + ":30000/app2"
        resp = requests.get(req_url, headers={"host": "test.media.com"})

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app2()
    
    def test_basic_routing_edge_app2_https(self, minikubeip):
        req_url = "https://" + minikubeip + ":30043/app2"
        resp = requests.get(req_url, headers={"host": "test.edge.com"}, verify=False)

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app2()
    
    def test_updating_ingress_media_app2(self, minikubeip):
        kubectl_apply('data/ats-ingress-update.yaml')
        req_url = "http://" + minikubeip + ":30000/app2"
        resp = requests.get(req_url, headers={"host": "test.media.com"})

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1_updated()
    
    def test_deleting_ingress_media_app2(self, minikubeip):
        kubectl_apply('data/ats-ingress-delete.yaml')
        req_url = "http://" + minikubeip + ":30000/app2"
        resp = requests.get(req_url, headers={"host": "test.media.com"})

        assert resp.status_code == 404,\
            f"Expected: 400 response code for test_basic_routing_deleted_ingress"

    def test_add_ingress_media(self, minikubeip):
        kubectl_apply('data/ats-ingress-add.yaml')
        req_url = "http://" + minikubeip + ":30000/test"
        resp = requests.get(req_url, headers={"host": "test.media.com"})

        assert resp.status_code == 200,\
            f"Expected: 200 response code for test_basic_routing"
        assert ' '.join(resp.text.split()) == get_expected_response_app1()

    def test_snippet_edge_app2(self, minikubeip):
        kubectl_apply('data/ats-ingress-snippet.yaml')
        req_url = "http://" + minikubeip + ":30000/app2"
        resp = requests.get(req_url, headers={"host": "test.edge.com"},allow_redirects=False)

        assert resp.status_code == 301,\
            f"Expected: 301 response code for test_snippet_edge_app2"
        assert resp.headers['Location'] == 'https://test.edge.com/app2'
    
        