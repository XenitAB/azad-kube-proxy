# azad-kube-proxy
Azure AD Kubernetes API Proxy

## Status

[![Coverage Status](https://coveralls.io/repos/github/XenitAB/azad-kube-proxy/badge.svg?branch=main)](https://coveralls.io/github/XenitAB/azad-kube-proxy?branch=main)

## Description

*ALPHA* project. Use at own risk.

This reverse proxy will run in front of a Kubernetes API and accept tokens from Azure AD and using these and the Graph API, use impersonation headers to authenticate the end user to the API.

## Overview

![overview](assets/azad-kube-proxy-overview.png)

## Why was this built?

There are a few reasons why this proxy was built, mainly:

- Azure AD authentication works great with Azure Kubernetes Service, but not that well with on-prem or other providers.
  - When a user is member of more than 200 groups, distributed claims will be used. Azure AD doesn't follow the OIDC specification for distributed claims which means it doesn't work by default.
    - Support OIDC distributed claims for group resolution in the K8S apiserver OIDC token checker [#62920](https://github.com/kubernetes/kubernetes/issues/62920)
    - [AppsCode Guard](https://github.com/appscode/guard/blob/master/auth/providers/azure/graph/aks_tokenprovider.go)
- When you do a blue/green deployment of Azure Kubernetes Service, a new API endpoint is used and everyone accessing the cluster will need to generate a new config. By using a proxy, you can use your own DNS records for it and just switch what cluster it is pointing to.
- Using the AKS Kubernetes API with service principals haven't been the easiest (and before AADv2 support, wasn't possible).
- Full control of the Azure AD Application that is published.
- Ability to filter groups based on prefix to only allow specific groups to be used with the cluster.
- Ability to create RBAC rules based on group names instead of ObjectIDs.

## What are the main features?

The main features of this proxy are:

- Use Azure AD authentication with all cloud providers and on-prem.
- Enable blue/green deployment of clusters.
- Resolve the distributed claims issue with Azure AD and Kubernetes.
- Limit what groups are sent to the Kubernetes API based on group name prefix.
- Ability to control the Azure AD Application fully.
- Use `az` (azure cli) to get tokens.
- Use both normal users and service principals.

## Alternatives

The following alternatives exists:

- [kube-oidc-proxy](https://github.com/jetstack/kube-oidc-proxy)
- [Pomerium](https://github.com/pomerium/pomerium)

## Local development

### Creating the Azure AD Application

```shell
AZ_APP_NAME="k8s-api"
AZ_APP_URI="https://k8s-api.azadkubeproxy.onmicrosoft.com"
AZ_APP_ID=$(az ad app create --display-name ${AZ_APP_NAME} --identifier-uris ${AZ_APP_URI} --query appId -o tsv)
AZ_APP_OBJECT_ID=$(az ad app show --id ${AZ_APP_ID} --output tsv --query objectId)
AZ_APP_PERMISSION_ID=$(az ad app show --id ${AZ_APP_ID} --output tsv --query "oauth2Permissions[0].id" )
az ad app update --id ${AZ_APP_ID} --set groupMembershipClaims=All
az rest --method PATCH --uri "https://graph.microsoft.com/beta/applications/${AZ_APP_OBJECT_ID}" --body '{"api":{"requestedAccessTokenVersion": 2}}'
# Add Azure CLI as allowed client
az rest --method PATCH --uri "https://graph.microsoft.com/beta/applications/${AZ_APP_OBJECT_ID}" --body "{\"api\":{\"preAuthorizedApplications\":[{\"appId\":\"04b07795-8ddb-461a-bbee-02f9e1bf7b46\",\"permissionIds\":[\"${AZ_APP_PERMISSION_ID}\"]}]}}"
AZ_APP_SECRET=$(az ad sp credential reset --name ${AZ_APP_ID} --credential-description "azad-kube-proxy" --output tsv --query password)
az ad app permission add --id ${AZ_APP_ID} --api 00000002-0000-0000-c000-000000000000 --api-permissions 5778995a-e1bf-45b8-affa-663a9f3f4d04=Role
az ad app permission add --id ${AZ_APP_ID} --api 00000003-0000-0000-c000-000000000000 --api-permissions 7ab1d382-f21e-4acd-a863-ba3e13f7da61=Role
az ad app permission admin-consent --id ${AZ_APP_ID}
```

### Generating self signed certificate for development

*NOTE*: You need to run the application using certificates since `kubectl` won't send Authorization header when not using TLS.

```shell
mkdir tmp
cd tmp

openssl req -newkey rsa:4096 \
            -x509 \
            -sha256
            -days 3650 \
            -nodes \
            -out tmp/cert.crt \
            -keyout tmp/cert.key \
            -subj "/C=SE/ST=LOCALHOST/L=LOCALHOST/O=LOCALHOST/OU=LOCALHOST/CN=localhost"

CERT_PATH="${PWD}/tmp/cert.crt"
KEY_PATH="${PWD}/tmp/cert.key"
```

### Setting up Kind cluster

```shell
kind create cluster --name azad-kube-proxy
CLUSTER_URL=$(kubectl config view --output json | jq -r '.clusters[] | select(.name | test("kind-azad-kube-proxy")).cluster.server')
HOST_PORT=$(echo ${CLUSTER_URL} | sed -e "s|https://||g")
K8S_HOST=$(echo ${HOST_PORT} | awk -F':' '{print $1}')
K8S_PORT=$(echo ${HOST_PORT} | awk -F':' '{print $2}')
```

### Configuring service account

```shell
kubectl config set-context kind-azad-kube-proxy
kubectl apply -f deploy/yaml/azad-kube-proxy.yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: temp
  namespace: azad-kube-proxy
spec:
  serviceAccountName: azad-kube-proxy
  containers:
  - image: busybox
    name: test
    command: ["sleep"]
    args: ["3000"]
EOF
kubectl exec -n azad-kube-proxy temp -- cat "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt" > tmp/ca.crt
kubectl exec -n azad-kube-proxy temp -- cat "/var/run/secrets/kubernetes.io/serviceaccount/token" > tmp/token
kubectl delete -n azad-kube-proxy pod temp
KUBE_CA_PATH="${PWD}/tmp/ca.crt"
KUBE_TOKEN_PATH="${PWD}/tmp/token"
```

### Creating env for tests

#### Creating test user / service principal

*PLEASE OBSERVE*: The below is an Azure AD tenant created for testing this proxy and nothing else. Don't use a production tenant for testing purposes.

```shell
USER_PASSWORD=$(base64 < /dev/urandom | tr -d 'O0Il1+/' | head -c 44; printf '\n')
USER_UPN="test.user@azadkubeproxy.onmicrosoft.com"
USER_NAME="test user"
USER_OBJECT_ID=$(az ad user create --display-name ${USER_NAME} --user-principal-name ${USER_UPN} --password ${USER_PASSWORD} --output tsv --query objectId)
SP_NAME="test-sp"
SP_CLIENT_ID=$(az ad app create --display-name ${SP_NAME} --output tsv --query appId)
SP_OBJECT_ID=$(az ad sp create --id ${SP_CLIENT_ID} --output tsv --query objectId)
SP_CLIENT_SECRET=$(az ad sp credential reset --name ${SP_CLIENT_ID} --credential-description ${SP_NAME} --years 10 --output tsv --query password) 

for i in `seq 1 10`; do
    echo "Run #${i}"
    PREFIX1_NAME="prefix1-${i}"
    PREFIX2_NAME="prefix2-${i}"
    az ad group create --display-name ${PREFIX1_NAME} --mail-nickname ${PREFIX1_NAME} 1>/dev/null
    az ad group create --display-name ${PREFIX2_NAME} --mail-nickname ${PREFIX2_NAME} 1>/dev/null
    az ad group member add --group ${PREFIX1_NAME} --member-id ${USER_OBJECT_ID} 1>/dev/null
    az ad group member add --group ${PREFIX2_NAME} --member-id ${USER_OBJECT_ID} 1>/dev/null
    az ad group member add --group ${PREFIX1_NAME} --member-id ${SP_OBJECT_ID} 1>/dev/null
    az ad group member add --group ${PREFIX2_NAME} --member-id ${SP_OBJECT_ID} 1>/dev/null
done
```

#### Creating env file for tests

```shell
echo "CLIENT_ID=${AZ_APP_ID}" > ${PWD}/tmp/test_env
echo "CLIENT_SECRET=${AZ_APP_SECRET}" >> ${PWD}/tmp/test_env
echo "TENANT_ID=$(az account show --output tsv --query tenantId)" >> ${PWD}/tmp/test_env
echo "TEST_USER_SP_CLIENT_ID=${SP_CLIENT_ID}" >> ${PWD}/tmp/test_env
echo "TEST_USER_SP_CLIENT_SECRET=${SP_CLIENT_SECRET}" >> ${PWD}/tmp/test_env
echo "TEST_USER_SP_RESOURCE=${AZ_APP_URI}" >> ${PWD}/tmp/test_env
echo "TEST_USER_SP_OBJECT_ID=${SP_OBJECT_ID}" >> ${PWD}/tmp/test_env
echo "TEST_USER_OBJECT_ID=${USER_OBJECT_ID}" >> ${PWD}/tmp/test_env
echo "TEST_USER_PASSWORD=${USER_PASSWORD}" >> ${PWD}/tmp/test_env
echo "AZURE_AD_GROUP_PREFIX=prefix1" >> ${PWD}/tmp/test_env
echo "KUBERNETES_API_HOST=${K8S_HOST}" >> ${PWD}/tmp/test_env
echo "KUBERNETES_API_PORT=${K8S_PORT}" >> ${PWD}/tmp/test_env
echo "KUBERNETES_API_CA_CERT_PATH=${KUBE_CA_PATH}" >> ${PWD}/tmp/test_env
echo "KUBERNETES_API_TOKEN_PATH=${KUBE_TOKEN_PATH}" >> ${PWD}/tmp/test_env
echo "TLS_ENABLED=true" >> ${PWD}/tmp/test_env
echo "TLS_CERTIFICATE_PATH=${CERT_PATH}" >> ${PWD}/tmp/test_env
echo "TLS_KEY_PATH=${KEY_PATH}" >> ${PWD}/tmp/test_env
echo "PORT=8443" >> ${PWD}/tmp/test_env
```

### Running the proxy

```shell
make run
```

### Authentication for end user

#### Curl

```shell
TOKEN=$(make token)
curl -k -H "Authorization: Bearer ${TOKEN}" https://localhost:8443/api/v1/namespaces/default/pods
```

#### Kubectl

```shell
TOKEN=$(make token)
kubectl --token="${TOKEN}" --server https://127.0.0.1:8443 --insecure-skip-tls-verify get pods
```