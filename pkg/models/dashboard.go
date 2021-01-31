package models

import "fmt"

// Dashboard ...
type Dashboard string

// NoneDashboard ...
var NoneDashboard Dashboard = "NONE"

// K8sdashDashboard ...
var K8sdashDashboard Dashboard = "K8SDASH"

// GetDashboard ...
func GetDashboard(s string) (Dashboard, error) {
	switch s {
	case "NONE":
		return NoneDashboard, nil
	case "K8SDASH":
		return K8sdashDashboard, nil
	default:
		return "", fmt.Errorf("Unknown dashboard '%s'. Supported engines are: NONE or K8SDASH", s)
	}
}
