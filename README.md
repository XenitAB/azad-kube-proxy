# azad-kube-proxy
Azure AD Kubernetes API Proxy

## Description

*ALPHA* project. Use at own risk.

This reverse proxy will run in front of a Kubernetes API and accept tokens from Azure AD and using these and the Graph API, use impersonation headers to authenticate the end user to the API.

## Local development

### Creating the Azure AD Application

```shell
AZ_APP_NAME="k8s-api-dev"
AZ_APP_URI="https://k8s-api.dev.xenit.io"
AZ_APP_ID=$(az ad app create --display-name ${AZ_APP_NAME} --identifier-uris ${AZ_APP_URI} --query appId -o tsv)
AZ_APP_OBJECT_ID=$(az ad app show --id ${AZ_APP_ID} --output tsv --query objectId)
AZ_APP_PERMISSION_ID=$(az ad app show --id ${AZ_APP_ID} --output tsv --query "oauth2Permissions[0].id" )
az ad app update --id ${AZ_APP_ID} --set groupMembershipClaims=All
az rest --method PATCH --uri "https://graph.microsoft.com/beta/applications/${AZ_APP_OBJECT_ID}" --body '{"api":{"requestedAccessTokenVersion": 2}}'
# Add Azure CLI as allowed client
az rest --method PATCH --uri "https://graph.microsoft.com/beta/applications/${AZ_APP_OBJECT_ID}" --body "{\"api\":{\"preAuthorizedApplications\":[{\"appId\":\"04b07795-8ddb-461a-bbee-02f9e1bf7b46\",\"permissionIds\":[\"${AZ_APP_PERMISSION_ID}\"]}]}}"
AZ_APP_SECRET=$(az ad sp credential reset --name ${AZ_APP_ID} --credential-description "azad-kube-proxy" --output tsv --query password)
az ad app permission add --id ${AZ_APP_ID} --api 00000002-0000-0000-c000-000000000000 --api-permissions 5778995a-e1bf-45b8-affa-663a9f3f4d04=Role
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
HOST=$(echo ${HOST_PORT} | awk -F':' '{print $1}')
PORT=$(echo ${HOST_PORT} | awk -F':' '{print $2}')
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

### Running the application

```shell
export CLIENT_ID=${AZ_APP_ID}
export CLIENT_SECRET=${AZ_APP_SECRET}
export TENANT_ID=$(az account show --output tsv --query tenantId)
export AZURE_AD_GROUP_PREFIX=""
export KUBERNETES_API_HOST=${HOST}
export KUBERNETES_API_PORT=${PORT}
export KUBERNETES_API_CA_CERT_PATH=${KUBE_CA_PATH}
export KUBERNETES_API_TOKEN_PATH=${KUBE_TOKEN_PATH}
export TLS_ENABLED="true"
export TLS_CERTIFICATE_PATH=${CERT_PATH}
export TLS_KEY_PATH=${KEY_PATH}
export PORT="8443"

go run cmd/azad-kube-proxy/main.go
```

### Authentication for end user

#### Curl

```shell
TOKEN=$(az account get-access-token --resource ${AZ_APP_URI} --query accessToken --output tsv)
curl -k -H "Authorization: Bearer ${TOKEN}" https://localhost:8443/api/v1/namespaces/default/pods
```

#### Kubectl

```shell
TOKEN=$(az account get-access-token --resource ${AZ_APP_URI} --query accessToken --output tsv)
kubectl --token="${TOKEN}" --server https://127.0.0.1:8443 --insecure-skip-tls-verify get pods
```