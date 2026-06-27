package hash

import (
	"strings"
	"testing"
)

func TestHashFileToSHA256(t *testing.T) {
	gotHash, gotSize, err := HashFileToSHA256(strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("HashFileToSHA256 returned error: %v", err)
	}
	if gotSize != 5 {
		t.Fatalf("size = %d, want 5", gotSize)
	}
	const wantHash = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if gotHash != wantHash {
		t.Fatalf("hash = %s, want %s", gotHash, wantHash)
	}
}
