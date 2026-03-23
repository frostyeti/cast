package cmd

import "testing"

func TestHandleToolInstallSpecialCases(t *testing.T) {
	originalInstallDeno := installDenoFunc
	originalInstallBun := installBunFunc
	originalInstallMise := installMiseFunc
	originalRunMise := runMiseCmdFunc
	defer func() {
		installDenoFunc = originalInstallDeno
		installBunFunc = originalInstallBun
		installMiseFunc = originalInstallMise
		runMiseCmdFunc = originalRunMise
	}()

	called := ""
	installDenoFunc = func() error {
		called = "deno"
		return nil
	}
	installBunFunc = func() error {
		called = "bun"
		return nil
	}
	installMiseFunc = func() error {
		called = "mise"
		return nil
	}
	runMiseCmdFunc = func(args []string) error {
		called = "mise-cmd"
		return nil
	}

	if err := handleToolInstall(toolInstallCmd, []string{"deno"}); err != nil {
		t.Fatalf("handleToolInstall(deno) returned error: %v", err)
	}
	if called != "deno" {
		t.Fatalf("expected deno installer to run, got %s", called)
	}

	if err := handleToolInstall(toolInstallCmd, []string{"bun"}); err != nil {
		t.Fatalf("handleToolInstall(bun) returned error: %v", err)
	}
	if called != "bun" {
		t.Fatalf("expected bun installer to run, got %s", called)
	}

	if err := handleToolInstall(toolInstallCmd, []string{"mise"}); err != nil {
		t.Fatalf("handleToolInstall(mise) returned error: %v", err)
	}
	if called != "mise" {
		t.Fatalf("expected mise installer to run, got %s", called)
	}

	if err := handleToolInstall(toolInstallCmd, []string{"python"}); err != nil {
		t.Fatalf("handleToolInstall(default) returned error: %v", err)
	}
	if called != "mise-cmd" {
		t.Fatalf("expected fallback to mise command, got %s", called)
	}
}
