package proxy

import (
	"encoding/json"
	"fmt"
)

type userModelType string

var normalUserModelType userModelType = "NormalUser"

var servicePrincipalUserModelType userModelType = "ServicePrincipal"

type userModel struct {
	Username string
	ObjectID string
	Groups   []groupModel
	Type     userModelType
}

func (i userModel) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

func (i *userModel) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &i)
}

type groupIdentifier string

var nameGroupIdentifier groupIdentifier = "NAME"

var objectIDGroupIdentifier groupIdentifier = "OBJECTID"

func GetGroupIdentifier(s string) (groupIdentifier, error) {
	switch s {
	case "NAME":
		return nameGroupIdentifier, nil
	case "OBJECTID":
		return objectIDGroupIdentifier, nil
	default:
		return "", fmt.Errorf("Unknown group identifier '%s'. Supported identifiers are: NAME or OBJECTID", s)
	}
}

type groupModel struct {
	Name     string
	ObjectID string
}

func (i groupModel) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

func (i *groupModel) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &i)
}
