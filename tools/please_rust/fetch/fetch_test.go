package fetch

import (
	"testing"
)

func TestParseBuildFile(t *testing.T) {
	content := `
rust_crate(
    name = "serde",
    version = "1.0.190",
)

rust_crate(
    name = "clap",
    version = "4.4.7",
)
`
	crates, err := ParseBuildFile(content)
	if err != nil {
		t.Fatalf("ParseBuildFile failed: %v", err)
	}

	if len(crates) != 2 {
		t.Fatalf("expected 2 crates, got %d", len(crates))
	}

	if crates[0].Name != "serde" || crates[0].Version != "1.0.190" {
		t.Errorf("unexpected crate 0: %+v", crates[0])
	}
	if crates[1].Name != "clap" || crates[1].Version != "4.4.7" {
		t.Errorf("unexpected crate 1: %+v", crates[1])
	}
}

func TestGenerateCargoToml(t *testing.T) {
	crates := []CrateReq{
		{Name: "serde", Version: "1.0.190"},
		{Name: "tokio", Version: "1.0", Features: []string{"full"}},
	}
	toml := GenerateCargoToml(crates)
	if toml == "" {
		t.Errorf("expected non-empty Cargo.toml string")
	}
}
