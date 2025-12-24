# Basic HTTPS Request Setup
Before testing SNI policies, a secure TLS setup is required. This involves:

## 1. Certificates Required

- Root CA Certificate: The Root Certificate Authority is the trust anchor. It signs all other certificates (host, client, backend) so they can be verified. Clients like curl use the Root CA to confirm the server certificate is trusted.
- Host certificate and key: Installed inside the ATS pod to terminate TLS.
- Client certificate and key: Required for client verification modes like moderate or strict.
- Backend Certificate: Used by ATS when connecting to backend services over TLS, ensuring secure communication and backend identity verification.



## 2. Why These Are Needed

- Root CA: Ensures the client trusts the server certificate by acting as the trust anchor.
- Host Certificate: Allows ATS to present a valid identity during the TLS handshake.
- Client Certificate: If client verification is enabled, ATS validates the client certificate against the CA to ensure mutual TLS.
- Backend Certificate: Ensures ATS can securely connect to backend services and verify their identity.

## 3. TLS Handshake Flow

- Client sends SNI (hostname) during TLS handshake.
- ATS responds with the correct certificate for that hostname.
- If verification policies are enabled, ATS checks client/server certificates accordingly.

## Different Backend Apps used.



| Application | Host | Description |
|---|---|---|
| app2 | test.edge.com | This is for HTTP, so no backend certificates required |
| node-app4 | test.example.com | Contains self signed backend certificates |
| node-app3 | test.example.com | Contains backend certificates signed by rootCA.|

#### The host certificates for node-app3 & node-app4 are mounted inside their pods. 

### Basic Curl Request
```
curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2
```
- --cacert certs/rootCA.crt: Trust the CA that signed the server certificate.

- --resolve: Map hostname to Minikube IP and port.
https://test.edge.com:30443/app2: HTTPS endpoint exposed by ATS ingress
- app2 is the backend application endpoint that ATS is routing traffic to.
 Components Explained

- --cert certs/client1.crt:  Provides the client certificate for mutual TLS (mTLS).
This proves the client’s identity to the server.

- --key certs/client1.key: Private key corresponding to client1.crt.
Required to complete the client authentication process.


### This is our expected response 
```
* Added test.edge.com:30443:10.63.20.30 to DNS cache
* Hostname test.edge.com was found in DNS cache
*   Trying 10.63.20.30:30443...
* Connected to test.edge.com (10.63.20.30) port 30443
* ALPN: curl offers h2,http/1.1
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
*  CAfile: certs/rootCA.crt
*  CApath: /etc/ssl/certs
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
* TLSv1.3 (IN), TLS handshake, Request CERT (13):
* TLSv1.3 (IN), TLS handshake, Certificate (11):
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
* TLSv1.3 (IN), TLS handshake, Finished (20):
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.3 (OUT), TLS handshake, Certificate (11):
* TLSv1.3 (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384 / X25519 / RSASSA-PSS
* ALPN: server accepted h2
* Server certificate:
*  subject: C=US; ST=State; L=City; O=MyOrg; OU=MyUnit; CN=test.edge.com
*  start date: Nov 27 05:42:34 2025 GMT
*  expire date: Nov 27 05:42:34 2026 GMT
*  subjectAltName: host "test.edge.com" matched cert's "test.edge.com"
*  issuer: C=US; ST=State; L=City; O=MyOrg; OU=MyUnit; CN=TestRootCA
*  SSL certificate verify ok.
*   Certificate level 0: Public key type RSA (2048/112 Bits/secBits), signed using sha256WithRSAEncryption
*   Certificate level 1: Public key type RSA (4096/152 Bits/secBits), signed using sha256WithRSAEncryption
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* old SSL session ID is stale, removing
* using HTTP/2
* [HTTP/2] [1] OPENED stream for https://test.edge.com:30443/app2
* [HTTP/2] [1] [:method: GET]
* [HTTP/2] [1] [:scheme: https]
* [HTTP/2] [1] [:authority: test.edge.com:30443]
* [HTTP/2] [1] [:path: /app2]
* [HTTP/2] [1] [user-agent: curl/8.5.0]
* [HTTP/2] [1] [accept: */*]
> GET /app2 HTTP/2
> Host: test.edge.com:30443
> User-Agent: curl/8.5.0
> Accept: */*
> 
< HTTP/2 200 
< x-powered-by: Express
< accept-ranges: bytes
< cache-control: public, max-age=0
< last-modified: Tue, 23 Sep 2025 06:38:46 GMT
< content-type: text/html; charset=UTF-8
< content-length: 188
< date: Thu, 27 Nov 2025 05:49:50 GMT
< etag: W/"bc-199754bccf0"
< age: 0
< server: ATS/9.2.11
< 
<!DOCTYPE html>
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

</HTML>
* Connection #0 to host test.edge.com left intact
```
# The policies which have been tested are listed below



# 1. HTTP/2 Enabled & Disabled Test
## I. Enabled
- When HTTP/2 is enabled in the SNI Policy, ATS should negotiate HTTP/2 during the TLS handshake.

-  Apply HTTP/2 Enabled Policy
```
kubectl apply -f ats_sni/http2/on.yaml
```
After applying the policy, the next step is to make the request to obtain the response.

```
curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2
```
Response:
```
* SSL connection using TLSv1.3
* ALPN, server accepted to use h2
> GET /app2 HTTP/2
< HTTP/2 200
```
This Shows:
- ALPN, server accepted to use h2 → HTTP/2 negotiated.
- Response status line: HTTP/2 200.

## II. Disabled

When HTTP/2 is disabled in the SNI Policy, ATS should NOT negotiate HTTP/2 during the TLS handshake. It should fall back to HTTP/1.1.


Apply HTTP/2 Disabled Policy
```
kubectl apply -f ats_sni/http2/off.yaml
```
Make the Request
```
curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2
```
Expected Response
```
* SSL connection using TLSv1.3
* ALPN, server accepted to use http/1.1
> GET /app2 HTTP/1.1
< HTTP/1.1 200 OK
```

This Shows:

- ALPN, server accepted to use http/1.1 → HTTP/2 was NOT negotiated.
- Response status line: HTTP/1.1 200 OK → Confirms fallback to HTTP/1.1.


# 2. Verify Client Policy 
## I. Strict Mode
When verifyClient is set to strict, ATS requires the client to present a valid certificate during the TLS handshake. If the client does not provide a certificate, the connection fails.

Apply Strict Policy
```
kubectl apply -f ats_sni/verify-client/strict.yaml
```

Curl Without Client Certificate
```
curl --cacert certs/rootCA.crt -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2
```

Expected Response:
```
* SSL connection using TLSv1.3
* error: tlsv13 alert certificate required
curl: (35) error: tlsv13 alert certificate required
```
This indicates:

- TLS handshake failed because ATS enforced client certificate verification.
- No HTTP response is returned.


Curl With Client Certificate
```
curl --cacert certs/rootCA.crt --cert certs/client1.crt --key certs/client1.key -v --resolve test.edge.com:30443:{minikubeip} https://test.edge.com:30443/app2
```



Expected Response:
```
* SSL connection using TLSv1.3
> GET /app2 HTTP/2
< HTTP/2 200
```
This confirms:

- TLS handshake succeeded with client certificate.
- HTTP/2 or HTTP/1.1 response based on policy.

Other Modes:
- none → No client certificate required; all requests succeed.
- moderate → Client certificate requested but optional; connection succeeds even without cert.

# 3. Host SNI Policy
Host SNI Policy determines how ATS handles mismatches between the SNI hostname (sent during TLS handshake) and the HTTP Host header.
Modes in Brief:

- disabled → ATS ignores mismatches; all requests succeed.
- enforced → ATS blocks mismatches with 403 Forbidden.
- permissive → ATS allows mismatches but logs them; request may return 404.


For verifying the Host SNI policy, the node-app3 application is used as the backend

## I. Disabled Mode

Apply Policy
```
kubectl apply -f ats_sni/host-sni-policy/disabled.yaml
```

Curl with mismatched SNI and Host:
```
curl --cacert certs/rootCA.crt -v \
  --resolve test.example.com:30443:{minikubeip} \
  https://test.example.com:30443/node-app3 \
  -H "Host: test.edge.com"
```

Expected Response:
```
* SSL connection using TLS
> GET /node-app3 HTTP/1.1
< HTTP/1.1 200 OK
```
Mismatch is ignored.

## II. Enforced Mode

Apply policy:
```
kubectl apply -f ats_sni/host-sni-policy/enforced.yaml
```
Curl with mismatch:
```
curl --cacert certs/rootCA.crt -v \
--resolve test.example.com:30443:{minikubeip} \
https://test.example.com:30443/node-app3 \
-H "Host: test.edge.com"
```

Expected Response:
```
* SSL connection using TLS
> GET /node-app3 HTTP/2
< HTTP/2 403
```
Check ATS logs:
```
kubectl exec <trafficserver-pod> -- \grep 'SNI/hostname mismatch' /opt/ats/var/log/trafficserver/diags.log
```
Log entry:
```
SNI/hostname mismatch sni=test.example.com host=test.edge.com action=terminate
```
## III. Permissive Mode
Apply policy:
```
kubectl apply -f ats_sni/host-sni-policy/permissive.yaml
```
Curl with mismatch:
```
curl --cacert certs/rootCA.crt -v \
--resolve test.example.com:30443:{minikubeip} \
https://test.example.com:30443/node-app3 \
-H "Host: test.edge.com"
```

Expected Response:
```
* SSL connection using TLS
> GET /node-app3 HTTP/2
< HTTP/2 404
```
**Reason for 404:**
The Host header (test.edge.com) does not match the SNI policy or routing configuration for test.example.com, so ATS cannot find a matching route and returns 404 Not Found.

Check ATS logs:
```
kubectl exec <trafficserver-pod> -- \grep 'SNI/hostname mismatch' /opt/ats/var/log/trafficserver/diags.log
```
Log entry:
```
SNI/hostname mismatch sni=test.example.com host=test.edge.com action=continue
```

# 4. Verify Server Policy

This policy controls how ATS validates the backend server’s certificate when making TLS connections to origin servers.

Modes in Brief:
- disabled → ATS does not verify backend certificates; all connections succeed.
- enforced → ATS strictly verifies backend certificates; invalid certs cause 502 Bad Gateway.
- permissive → ATS allows invalid certs but logs warnings; request still succeeds.

For verifying the server policy, the node-app3 application is used as the backend, and it has valid backend certificates to ensure secure TLS communication.

## I. Enforced Mode
Apply policy:
```
kubectl apply -f ats_sni/verify-server-policy/enforced.yaml
```

Valid Backend Certificate
```
curl --cacert certs/rootCA.crt -v \
--resolve test.example.com:30443:{minikubeip} \
https://test.example.com:30443/node-app3
```

Expected Response:
```
* SSL connection using TLS
< HTTP/2 200
```

## Invalid Backend Certificate

In this the backend certificates provided in node-app4 are self signed hence the  verification will fail and response will show error 502 Bad Gateway.

```
curl --cacert certs/rootCA.crt -v \
--resolve test.example.com:30443:{minikubeip} \
https://test.example.com:30443/node-app4
```

Expected Response:
```
* SSL connection using TLS
< HTTP/2 502
```
Check logs:
```
kubectl exec <trafficserver-pod> -- \grep 'Action=Terminate' /opt/ats/var/log/trafficserver/diags.log
```


Log entry:
```
Core server certificate verification failed Action=Terminate
```

## II. Disabled Mode
Apply policy:
```
kubectl apply -f ats_sni/verify-server-policy/disabled.yaml
```

Curl with invalid cert:
```
curl --cacert certs/rootCA.crt -v \
--resolve test.example.com:30443:{minikubeip} \
https://test.example.com:30443/node-app4
```

Expected Response:
```
* SSL connection using TLS
< HTTP/2 200
```

ATS ignores certificate validation.

## III. Permissive Mode
Apply policy:
```
kubectl apply -f ats_sni/verify-server-policy/permissive.yaml 
```

Curl with invalid cert:
```
curl --cacert certs/rootCA.crt -v \
--resolve test.example.com:30443:{minikubeip} \
https://test.example.com:30443/node-app4
```

Expected Response:
```
* SSL connection using TLS
< HTTP/2 200
```

Check logs for warnings:
```
kubectl exec <trafficserver-pod> -- \grep 'Core server certificate verification failed' /opt/ats/var/log/trafficserver/diags.log
```

Log entry:
```
Core server certificate verification failed Action=Continue
```

### All configured tests have been executed successfully. Each scenario behaved as expected according to the applied policies, validating correct functionality across all test cases.









