package httpclient

import "testing"

func TestTruncate(t *testing.T) {
	if Truncate("short", 10) != "short" {
		t.Error("short string should not be truncated")
	}
	result := Truncate("this is a long string", 10)
	if result != "this is a ..." {
		t.Errorf("truncate: got %q", result)
	}
}
