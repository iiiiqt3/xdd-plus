package models

import "testing"

func TestCanAccessPortalContent(t *testing.T) {
	if CanAccessPortalContent(1, 1000) != true {
		t.Fatal("coin 1000 should allow access")
	}
	if CanAccessPortalContent(1, 999) != false {
		t.Fatal("coin 999 without monthly project should deny")
	}
}

func TestPortalMinCoinZeroMeansOpen(t *testing.T) {
	zero := 0
	old := sysConfig.PortalMinCoinForAccess
	sysConfig.PortalMinCoinForAccess = &zero
	defer func() { sysConfig.PortalMinCoinForAccess = old }()

	if GetPortalMinCoinForAccess() != 0 {
		t.Fatalf("required=%d want 0", GetPortalMinCoinForAccess())
	}
	if !CanAccessPortalContent(1, 0) {
		t.Fatal("required 0 should allow any coin")
	}
	info := BuildPortalAccessInfo(0, 0)
	if !info.Allowed {
		t.Fatal("expected allowed when required is 0")
	}
}

func TestBuildPortalAccessInfo(t *testing.T) {
	info := BuildPortalAccessInfo(0, 320)
	if info.Allowed {
		t.Fatal("expected denied")
	}
	if info.GapCoin != 680 {
		t.Fatalf("gap=%d want 680", info.GapCoin)
	}
	if info.Message == "" {
		t.Fatal("expected message")
	}
}
