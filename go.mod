module github.com/xenitab/azad-kube-proxy

go 1.16

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v0.16.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v0.8.0
	github.com/alicebob/miniredis/v2 v2.14.3
	github.com/bombsimon/logrusr v1.1.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/go-playground/validator/v10 v10.6.1
	github.com/go-redis/redis/v8 v8.8.2
	github.com/google/go-cmp v0.5.5
	github.com/gorilla/mux v1.8.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/manicminer/hamilton v0.12.0
	github.com/manifoldco/promptui v0.8.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pquerna/cachecontrol v0.0.0-20171018203845-0dec1b30a021 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/cli/v2 v2.3.0
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/square/go-jose.v2 v2.2.2 // indirect
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.0
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
)
