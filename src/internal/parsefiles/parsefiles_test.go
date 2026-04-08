package parsefiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTomlField_inlineCommentAndQuotes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "node.toml")
	content := `
# head
network_name = "testnet"                      # DO NOT CHANGE
chain_id = 10143
self_address = "51.255.93.32:8000"               # IP Address and Port
self_record_seq_num = 6                    # Should not leak
`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := TomlField(p, "network_name"); got != "testnet" {
		t.Fatalf("network_name: got %q", got)
	}
	if got := TomlField(p, "chain_id"); got != "10143" {
		t.Fatalf("chain_id: got %q", got)
	}
	if got := TomlField(p, "self_address"); got != "51.255.93.32:8000" {
		t.Fatalf("self_address: got %q", got)
	}
	if got := TomlField(p, "self_record_seq_num"); got != "6" {
		t.Fatalf("self_record_seq_num: got %q", got)
	}
}

func TestStripTomlLineComment(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`"a" # x`, `"a"`},
		{`foo # bar`, `foo`},
		{`"a # b" # c`, `"a # b"`},
	}
	for _, tc := range cases {
		if got := stripTomlLineComment(tc.in); got != tc.want {
			t.Errorf("stripTomlLineComment(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
