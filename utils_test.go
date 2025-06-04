package main

import "testing"

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"foo", "foo"},
		{"fooBar", "foo_bar"},
		{"fooBarBaz", "foo_bar_baz"},
		{"FooBar", "foo_bar"},
		{"FooBarBaz", "foo_bar_baz"},
		{"foo_bar", "foo_bar"},
		{"foo_bar_baz", "foo_bar_baz"},
		{"HTTPRequest", "http_request"},
	}
	for _, test := range tests {
		actual := ToSnakeCase(test.input)
		if actual != test.expected {
			t.Errorf("toSnakeCase(%q) = %q, want %q", test.input, actual, test.expected)
		}
	}
}
