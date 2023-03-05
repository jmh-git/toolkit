package toolkit

import (
	"strings"
	"testing"
)

func TestRandomStringStartsWithAlpha(t *testing.T) {
	tools := Tools{}

	n := 10
	s := tools.RandomStringWithAlpha(n)
	if len(s) != n {
		t.Errorf("Invalid string length - want %d, got %d\n", n, len(s))
	}

	allowedFirst := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, r := range s {
		if !strings.ContainsRune(allowedFirst, r) {
			t.Errorf("Random string doesn't start w/ alpha - got %v\n", r)
		}
		break
	}
}

func TestRanddomStringLength(t *testing.T) {
	tools := Tools{}

	n := 10
	s := tools.RandomString(n)
	if len(s) != n {
		t.Errorf("Invalid string length - want %d, got %d\n", n, len(s))
	}
}

func BenchmarkRandomStringWithAlphaShort(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomStringWithAlpha(10)
	}
}

func BenchmarkRandomStringWithAlphaLong(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomStringWithAlpha(100)
	}
}

func BenchmarkRandomStringShort(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomString(10)
	}
}

func BenchmarkRandomStringLong(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomString(100)
	}
}
