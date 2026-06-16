package bot

import "testing"

func TestParseSedCommand(t *testing.T) {
	cases := []struct {
		input     string
		wantPat   string
		wantRepl  string
		wantFlags string
		wantOK    bool
	}{
		{"s/foo/bar/", "foo", "bar", "", true},
		{"s/foo/bar/g", "foo", "bar", "g", true},
		{"s/foo/bar/gi", "foo", "bar", "gi", true},
		{"s|foo|bar|", "foo", "bar", "", true},
		{"s_foo_bar_", "foo", "bar", "", true},
		{"s:foo:bar:", "foo", "bar", "", true},
		{"hello", "", "", "", false},
		{"s/", "", "", "", false},
		{"s//bar/", "", "", "", false},
		{"s/foo/bar", "foo", "bar", "", true},
	}
	for _, c := range cases {
		pat, repl, flags, ok := parseSedCommand(c.input)
		if ok != c.wantOK {
			t.Errorf("parseSedCommand(%q) ok=%v want %v", c.input, ok, c.wantOK)
			continue
		}
		if ok {
			if pat != c.wantPat {
				t.Errorf("parseSedCommand(%q) pattern=%q want %q", c.input, pat, c.wantPat)
			}
			if repl != c.wantRepl {
				t.Errorf("parseSedCommand(%q) replacement=%q want %q", c.input, repl, c.wantRepl)
			}
			if flags != c.wantFlags {
				t.Errorf("parseSedCommand(%q) flags=%q want %q", c.input, flags, c.wantFlags)
			}
		}
	}
}
