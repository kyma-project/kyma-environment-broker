package broker

import (
	"encoding/json"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

func Marshal(obj interface{}) []byte {
	if obj == nil {
		return []byte{}
	}
	bytes, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return bytes
}

func ClearOIDCInput(oidc *pkg.OIDCConnectDTO) {
	if oidc != nil {
		if oidc.OIDCConfigDTO != nil && oidc.OIDCConfigDTO.RequiredClaims != nil {
			oidc.OIDCConfigDTO.RequiredClaims = nil
		}
		if oidc.OIDCConfigDTO != nil && oidc.OIDCConfigDTO.GroupsPrefix != "" {
			oidc.OIDCConfigDTO.GroupsPrefix = ""
		}
	}
}
