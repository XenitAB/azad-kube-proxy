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

### Running the application

```shell
export CLIENT_ID=${AZ_APP_ID}
export CLIENT_SECRET=${AZ_APP_SECRET}
export TENANT_ID=$(az account show --output tsv --query tenantId)
export AZURE_AD_GROUP_PREFIX="<prefix>"
export KUBERNETES_API_HOST="<ip / hostname>"
export KUBERNETES_API_PORT="<port>"
export KUBERNETES_API_CA_CERT_PATH="<ca cert>"
export KUBERNETES_API_TOKEN_PATH="<token file>"

go run cmd/azad-kube-proxy/main.go
```

### Authentication for end user

```shell
TOKEN=$(az account get-access-token --resource ${AZ_APP_URI} --query accessToken --output tsv)
curl -H "Authorization: Bearer ${TOKEN}" http://localhost:8080/api/v1/namespaces/default/pods
```