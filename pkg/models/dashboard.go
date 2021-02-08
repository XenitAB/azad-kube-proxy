package models

import "fmt"

// Dashboard ...
type Dashboard string

// NoneDashboard ...
var NoneDashboard Dashboard = "NONE"

// K8sdashDashboard ...
var K8sdashDashboard Dashboard = "K8DASH"

// GetDashboard ...
func GetDashboard(s string) (Dashboard, error) {
	switch s {
	case "NONE":
		return NoneDashboard, nil
	case "K8DASH":
		return K8sdashDashboard, nil
	default:
		return "", fmt.Errorf("Unknown dashboard '%s'. Supported engines are: NONE or K8DASH", s)
	}
}
