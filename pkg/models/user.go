package models

import "encoding/json"

// User is the struct for a currently logged in user
type User struct {
	Username string
	Groups   []Group
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

// Group is the struct for a group
type Group struct {
	Name string
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
