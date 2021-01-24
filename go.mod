module github.com/xenitab/azad-kube-proxy

go 1.15

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v0.14.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v0.8.0
	github.com/alicebob/miniredis/v2 v2.14.1
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.3.0
	github.com/go-playground/validator/v10 v10.4.1
	github.com/go-redis/redis/v8 v8.4.9
	github.com/google/go-cmp v0.5.4
	github.com/gorilla/mux v1.8.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/manicminer/hamilton v0.4.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pquerna/cachecontrol v0.0.0-20171018203845-0dec1b30a021 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/urfave/cli/v2 v2.3.0
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/square/go-jose.v2 v2.2.2 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.20.2
