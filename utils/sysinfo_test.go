package utils

import "testing"

func TestGetSeverStatus(t *testing.T) {
	ServerStatus := GetSeverStatus()
	t.Log(ServerStatus)
}
