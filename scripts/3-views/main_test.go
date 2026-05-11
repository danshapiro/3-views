package main

import "testing"

func TestDefaultPermConfigAllowsRepoInspectionCommands(t *testing.T) {
	cfg := defaultPermConfig("/tmp/3-views-test/scratch")

	if cfg.ExternalDirectory != "allow" {
		t.Fatalf("ExternalDirectory = %v, want allow", cfg.ExternalDirectory)
	}
	if cfg.Grep != "allow" {
		t.Fatalf("Grep = %v, want allow", cfg.Grep)
	}
	if cfg.Glob != "allow" {
		t.Fatalf("Glob = %v, want allow", cfg.Glob)
	}
	if cfg.Lsp != "allow" {
		t.Fatalf("Lsp = %v, want allow", cfg.Lsp)
	}

	for _, pattern := range []string{
		"git status*",
		"git diff*",
		"git log*",
		"git show*",
		"git rev-parse*",
		"git ls-files*",
		"grep *",
		"ls",
		"ls *",
		"head *",
		"tail *",
		"wc *",
		"pwd",
		"mktemp /tmp/3-views-test/scratch/*",
		"mktemp -d /tmp/3-views-test/scratch/*",
	} {
		if cfg.Bash[pattern] != "allow" {
			t.Errorf("Bash[%q] = %q, want allow", pattern, cfg.Bash[pattern])
		}
	}
}

func TestDefaultPermConfigAllowsEditsOnlyInScratchDir(t *testing.T) {
	cfg := defaultPermConfig("/tmp/3-views-test/scratch")

	edit, ok := cfg.Edit.(map[string]string)
	if !ok {
		t.Fatalf("Edit = %[1]T(%[1]v), want map[string]string", cfg.Edit)
	}

	if edit["/tmp/3-views-test/scratch/*"] != "allow" {
		t.Errorf("Edit scratch file rule = %q, want allow", edit["/tmp/3-views-test/scratch/*"])
	}
	if edit["/tmp/3-views-test/scratch/**"] != "allow" {
		t.Errorf("Edit scratch tree rule = %q, want allow", edit["/tmp/3-views-test/scratch/**"])
	}
	if edit["*"] != "deny" {
		t.Errorf("Edit catch-all rule = %q, want deny", edit["*"])
	}
}

func TestDefaultPermConfigReadRulesProtectEnvFiles(t *testing.T) {
	cfg := defaultPermConfig("/tmp/3-views-test/scratch")

	for pattern, want := range map[string]interface{}{
		"*":             "allow",
		"*.env":         "deny",
		"*.env.*":       "deny",
		"*.env.example": "allow",
	} {
		if cfg.Read[pattern] != want {
			t.Errorf("Read[%q] = %v, want %v", pattern, cfg.Read[pattern], want)
		}
	}
}

func TestDefaultPermConfigAvoidsKnownMutableBashPatterns(t *testing.T) {
	cfg := defaultPermConfig("/tmp/3-views-test/scratch")

	for _, pattern := range []string{
		"git branch*",
		"git branch --show-current",
		"find *",
		"cat *",
		"sed -n *",
		"rm *",
		"mv *",
		"cp *",
		"chmod *",
		"chown *",
		"curl *",
		"wget *",
		"ssh *",
		"scp *",
		"dd *",
		"python *",
		"node *",
		"bash *",
		"sh *",
		"mktemp*",
		"mktemp",
		"mktemp -d",
		"mktemp -t *",
		"mktemp /tmp/*",
		"mktemp ./scratch.*",
		"mktemp -p . scratch.*",
	} {
		if action, ok := cfg.Bash[pattern]; ok {
			t.Errorf("Bash[%q] = %q, want no rule", pattern, action)
		}
	}

	if cfg.Bash["*"] != "deny" {
		t.Fatalf("Bash[*] = %q, want deny", cfg.Bash["*"])
	}
}
