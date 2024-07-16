#!/bin/sh
set -e
set -x

APP="ransim"

SUBJECT="/C=GR/ST=Greece/L=Athens/O=INTRACOM S.A. TELECOM SOLUTIONS/OU=Software Development Center/CN=NFV-RI"
ADD_EXTENSIONS="subjectAltName=DNS:ran-simulator,DNS:*"

# Generate Certificate Authority Private Key
sudo openssl genrsa -out "${APP}CA".key 4096
# Generate Certificate Authority Certificate
sudo openssl req -new -x509 -key "${APP}CA".key -out "${APP}CA".crt -subj "$SUBJECT"
# Generate APP Private Key
sudo openssl genrsa -out "${APP}".key 2048
# Generate APP Certificate
sudo openssl req -x509 -new -CA "${APP}CA".crt -CAkey "${APP}CA".key -key "${APP}".key -out "${APP}".crt -days 3650 -subj "$SUBJECT" -addext "$ADD_EXTENSIONS"
# Verify APP Certificate
sudo openssl verify -CAfile "${APP}CA".crt "${APP}".crt
# Print details of APP Certificate
openssl x509 -text -noout -in "${APP}".crt 

exit $?





