package appreleases

import (
	"context"
	"testing"
)

func TestSystemManagementDemoMenuActionPersistsInOverview(t *testing.T) {
	service := NewService(Config{})
	ctx := context.Background()

	overview, err := service.SystemManagementOverview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}
	menuID := ""
	for _, menu := range overview.Menus {
		if menu.Path == "/system/menus" {
			menuID = menu.ID
			break
		}
	}
	if menuID == "" {
		t.Fatal("expected menu editor in demo overview")
	}

	if resp, err := service.SystemMenuAction(ctx, menuID, "hide"); err != nil {
		t.Fatalf("hide menu failed: %v", err)
	} else if resp.Menu.Visible {
		t.Fatal("expected hide action to return invisible menu")
	}

	overview, err = service.SystemManagementOverview(ctx)
	if err != nil {
		t.Fatalf("overview after hide failed: %v", err)
	}
	foundHidden := false
	for _, menu := range overview.Menus {
		if menu.ID == menuID {
			foundHidden = !menu.Visible
			break
		}
	}
	if !foundHidden {
		t.Fatal("expected demo overview to persist hidden menu state")
	}

	if resp, err := service.SystemMenuAction(ctx, menuID, "show"); err != nil {
		t.Fatalf("show menu failed: %v", err)
	} else if !resp.Menu.Visible {
		t.Fatal("expected show action to return visible menu")
	}
}

func TestSystemManagementDemoCreateUserPersistsInOverview(t *testing.T) {
	service := NewService(Config{})
	ctx := context.Background()

	resp, err := service.CreateSystemUser(ctx, CreateSystemUserRequest{
		Name:       "Smoke 用户",
		Account:    "smoke.user",
		RoleCode:   "release_viewer",
		Department: "平台工程",
		Status:     "enabled",
	})
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	overview, err := service.SystemManagementOverview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}
	for _, user := range overview.Users {
		if user.ID == resp.User.ID && user.Account == "smoke.user" {
			return
		}
	}
	t.Fatal("expected created demo user in overview")
}
