package models

// User is the struct for a currently logged in user
type User struct {
	Username string
	Groups   []Group
}

// Group is the struct for a group
type Group struct {
	Name string
}
