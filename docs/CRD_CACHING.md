# Caching CRD for ATS


## Before enabling the cache

Let us check how we can verify whether caching is happening or not using curl command:
```bash
curl -v -H "Host: test.edge.com" http://{minikubeip}:30080/app1
```
We need to use the respective ip of the minikube we are using or the node ip on which the ats ingress controller is running.

The response we receive has the following details along with the HTML response (output for the first curl command):
```bash
> GET /app1 HTTP/1.1
> Host: test.edge.com
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 200 OK
< X-Powered-By: Express
< Accept-Ranges: bytes
< Cache-Control: public, max-age=0
< Last-Modified: Fri, 18 Jul 2025 07:01:16 GMT
< Content-Type: text/html; charset=UTF-8
< Content-Length: 190
< Date: Wed, 03 Sep 2025 09:48:09 GMT
< Etag: W/"be-1981c565260"
< Age: 0
< Connection: keep-alive
< Server: ATS/9.2.11
```
Now, when we run the same command after 6 seconds, we will have a response which will have following details:
```bash
> GET /app1 HTTP/1.1
> Host: test.edge.com
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 200 OK
< X-Powered-By: Express
< Accept-Ranges: bytes
< Cache-Control: public, max-age=0
< Last-Modified: Fri, 18 Jul 2025 07:01:16 GMT
< Content-Type: text/html; charset=UTF-8
< Content-Length: 190
< Date: Wed, 03 Sep 2025 09:48:15 GMT
< Etag: W/"be-1981c565260"
< Age: 0
< Connection: keep-alive
< Server: ATS/9.2.11
```
When we observe the details for the response for both the curl executions, the value for `Age` is `0` and the `Date` field has different values (look for the seconds), indicating response was not cached.

## Enabling the cache

### Steps to take before applying caching CRD (needed only first time) :
To apply a file we use `kubectl apply -f <filename.yaml>`.
- Go to the folder `trafficserver-ingress-controller/ats_caching`.
- Apply the file `ats-cachingpolicy-role.yaml`.
- Apply the file `ats-cachingpolicy-binding.yaml`.

The `ats-cachingpolicy-role.yaml` file defines a cluster-wide role named `ats-cachingpolicy-role`, which grants read-only permissions (`get`, `list`, `watch`) on the `atscachingpolicies` resource within the `k8s.trafficserver.apache.com` API group.

The `ats-cachingpolicy-binding.yaml` file binds the `ats-cachingpolicy-role` cluster role to the `default` service account, which allows the pods running under the `default` service account to read and watch `ATSCachingPolicy` objects across the cluster.

### Steps for applying CRD to enable caching:
- Before applying the CRD check the currently available crds using `kubectl get crd`.
- Go to the folder `trafficserver-ingress-controller/ats_caching`
- Apply the file `crd-atscachingpolicy.yaml`.
- Apply the file `atscachingpolicy.yaml`.
- Now again check the available crds, `using kubectl get crd`.

We will notice a new crd:
```bash
NAME                                                  CREATED AT
atscachingpolicies.k8s.trafficserver.apache.com       2025-09-03T09:45:13Z

```
Which was not available earlier.


### The content of atscachingpolicy.yaml is:
```yaml
apiVersion: k8s.trafficserver.apache.com/v1
kind: ATSCachingPolicy
metadata:
  name: my-app-caching
  namespace: caching-ats-new
spec:
  rules:
    - name: home-endpoint
      primarySpecifier:
        type: url_regex
        pattern: ".*/app1"
      action: cache
      ttl: "12s"
```
Here, we have enabled cache for the pattern `.*app1` for `12` seconds. After `12` seconds of running the curl command the response won’t be available in the cache.

## After enabling the cache
Execute the curl command
```bash
curl -v -H "Host: test.edge.com" http://{minikubeip}:30080/app1
```
The response we receive has the following details along with the HTML response( if same command was not run few minutes earlier):
```bash
> GET /app1 HTTP/1.1
> Host: test.edge.com
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 200 OK
< X-Powered-By: Express
< Accept-Ranges: bytes
< Cache-Control: public, max-age=0
< Last-Modified: Fri, 18 Jul 2025 07:01:16 GMT
< Content-Type: text/html; charset=UTF-8
< Content-Length: 190
< Date: Wed, 03 Sep 2025 09:51:02 GMT
< Etag: W/"be-1981c565260"
< Age: 0
< Connection: keep-alive
< Server: ATS/9.2.11
```
Now, when we run the same command after 7 seconds we will have a response along with following details:
```bash
> GET /app1 HTTP/1.1
> Host: test.edge.com
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/1.1 200 OK
< X-Powered-By: Express
< Accept-Ranges: bytes
< Cache-Control: public, max-age=0
< Last-Modified: Fri, 18 Jul 2025 07:01:16 GMT
< Content-Type: text/html; charset=UTF-8
< Content-Length: 190
< Date: Wed, 03 Sep 2025 09:51:02 GMT
< Etag: W/"be-1981c565260"
< Age: 7
< Connection: keep-alive
< Server: ATS/9.2.11
```
Observe both curl execution details, in both cases the value for `Age` is different and the `Date` field has the same values (look for the seconds).



