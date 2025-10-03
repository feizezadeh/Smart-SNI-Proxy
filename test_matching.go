package main

import (
	"fmt"
	"strings"
)

func trimDot(s string) string { return strings.TrimSuffix(s, ".") }

func matches(host, pattern string) bool {
	h := strings.ToLower(trimDot(host))
	p := strings.ToLower(trimDot(pattern))
	if p == "" {
		return false
	}
	if strings.HasPrefix(p, "*.") {
		suf := p[1:]
		if !strings.HasSuffix(h, suf) {
			return false
		}
		prefixLen := len(h) - len(suf)
		return prefixLen > 0
	}
	return h == p
}

func main() {
	// Test cases
	tests := []struct {
		host    string
		pattern string
		want    bool
		desc    string
	}{
		// Exact matches
		{"example.com", "example.com", true, "Exact match"},
		{"example.com", "*.example.com", false, "Wildcard should NOT match bare domain"},

		// Wildcard matches
		{"www.example.com", "*.example.com", true, "Wildcard should match www"},
		{"api.example.com", "*.example.com", true, "Wildcard should match api"},
		{"cdn.api.example.com", "*.example.com", true, "Wildcard should match multi-level"},

		// Real world - whatismyipaddress.com
		{"whatismyipaddress.com", "whatismyipaddress.com", true, "Exact match whatismyipaddress.com"},
		{"whatismyipaddress.com", "*.whatismyipaddress.com", false, "Wildcard should NOT match bare whatismyipaddress.com"},
		{"www.whatismyipaddress.com", "*.whatismyipaddress.com", true, "Wildcard should match www.whatismyipaddress.com"},

		// Edge cases
		{"example.com.", "example.com", true, "Trailing dot should be ignored"},
		{"www.example.com.", "*.example.com", true, "Trailing dot in wildcard"},
	}

	fmt.Println("Testing domain matching logic:")
	fmt.Println("================================")

	passed := 0
	failed := 0

	for _, tt := range tests {
		result := matches(tt.host, tt.pattern)
		status := "✅ PASS"
		if result != tt.want {
			status = "❌ FAIL"
			failed++
		} else {
			passed++
		}

		fmt.Printf("%s | Host: %-30s Pattern: %-30s Want: %-5t Got: %-5t | %s\n",
			status, tt.host, tt.pattern, tt.want, result, tt.desc)
	}

	fmt.Println("================================")
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)

	if failed > 0 {
		fmt.Println("\n❌ TESTS FAILED")
	} else {
		fmt.Println("\n✅ ALL TESTS PASSED")
	}
}
