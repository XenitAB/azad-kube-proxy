package models

import (
	"encoding/json"
	"fmt"
)

// UserType is the type of user
type UserType string

// NormalUserType is a normal user
var NormalUserType UserType = "NormalUser"

// ServicePrincipalUserType is a serivce principal
var ServicePrincipalUserType UserType = "ServicePrincipal"

// User is the struct for a currently logged in user
type User struct {
	Username string
	ObjectID string
	Groups   []Group
	Type     UserType
}

// MarshalBinary to marshal User object
func (i User) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

// UnmarshalBinary to unmarshal User object
func (i *User) UnmarshalBinary(data []byte) error {
	// convert data to yours, let's assume its json data
	return json.Unmarshal(data, &i)
}

// GroupIdentifier is the type for what identifier should be sent to the backend
type GroupIdentifier string

// NameGroupIdentifier defines that the Name should be sent to the backend
var NameGroupIdentifier GroupIdentifier = "NAME"

// ObjectIDGroupIdentifier defines that the ObjectID should be sent to the backend
var ObjectIDGroupIdentifier GroupIdentifier = "OBJECTID"

// GetGroupIdentifier ...
func GetGroupIdentifier(s string) (GroupIdentifier, error) {
	switch s {
	case "NAME":
		return NameGroupIdentifier, nil
	case "OBJECTID":
		return ObjectIDGroupIdentifier, nil
	default:
		return "", fmt.Errorf("Unkown group identifier %s. Supported identifiers are: NAME or OBJECTID", s)
	}
}

// Group is the struct for a group
type Group struct {
	Name     string
	ObjectID string
}

// MarshalBinary to marshal Group object
func (i Group) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

// UnmarshalBinary to unmarshal Group object
func (i *Group) UnmarshalBinary(data []byte) error {
	// convert data to yours, let's assume its json data
	return json.Unmarshal(data, &i)
}
