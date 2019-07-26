#Generate test certificates using openssl 1.1.1

1. Install openssl v1.1.1
```command
brew install openssl@1.1
```
2. Generate the CA certificate key and your certificate key:
```command
openssl genrsa -out <ca-cert-name>.key 2048
openssl genrsa -out <certificate-name>.key 2048
```
3. Generate the CA certificate
```command
openssl req -x509 -new -nodes -key <ca-cert-name>.key -sha256 -days 3650 -out <ca-cert-name> -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=Acme Root CA"
```
4. Create the request config file `ca.conf`
```
[req]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = req_ext
[req_distinguished_name]
countryName                 = CN
stateOrProvinceName         = GD
organizationName           = Acme
commonName                 = Acme Root CA
[req_ext]
subjectAltName = @alt_names
[alt_names]
IP.1   = 127.0.0.1
[SAN]
subjectAltName=IP:127.0.0.1
```
5. Create the Certificate Signing Request 
```command
/usr/local/opt/openssl@1.1/bin/openssl req -new -sha256 -key <certificate-name>.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=Acme Root CA" -reqexts SAN -config ca.conf -out ca.csr
```
6. Sign the certificate
* Server certificate
```command
/usr/local/opt/openssl@1.1/bin/openssl x509 -req -in ca.csr -CA <ca-cert-name> -CAkey ca.key -CAcreateserial -out <certificate-name>.crt -days 3650 -sha256 -extfile ca.conf -extensions req_ext
```
* Client certificate
```command
/usr/local/opt/openssl@1.1/bin/openssl x509 -req -in ca.csr -CA <ca-cert-name> -CAkey ca.key -CAcreateserial -out <certificate-name>.crt -days 3650 -sha256 
```
