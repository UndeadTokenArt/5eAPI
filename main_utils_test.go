package main

import "testing"

func TestPrettyCat(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "hyphenated", in: "magic-items", want: "Magic Items"},
		{name: "single word", in: "spells", want: "Spells"},
		{name: "empty", in: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := prettyCat(tc.in)
			if got != tc.want {
				t.Fatalf("prettyCat(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIconFor(t *testing.T) {
	if got := iconFor("monsters"); got != "👹" {
		t.Fatalf("iconFor(monsters) = %q, want %q", got, "👹")
	}
	if got := iconFor("unknown-category"); got != "📌" {
		t.Fatalf("iconFor(unknown-category) = %q, want %q", got, "📌")
	}
}

func TestExtractProps(t *testing.T) {
	data := map[string]interface{}{
		"name":        "Ignored Name",
		"index":       "ignored-index",
		"url":         "/ignored-url",
		"desc":        []interface{}{"ignored desc"},
		"armor_class": 15.0,
		"legendary":   true,
		"source":      "MM",
		"damage": []interface{}{
			map[string]interface{}{"name": "Fire"},
			"Cold",
			2.0,
		},
		"bad_list": []interface{}{map[string]interface{}{"foo": "bar"}},
	}

	props := extractProps(data)
	if len(props) == 0 {
		t.Fatal("extractProps returned no properties")
	}

	if hasLabel(props, "Name") {
		t.Fatal("extractProps should skip reserved key: name")
	}

	if !hasValue(props, "15") {
		t.Fatalf("expected a property value of %q, got %#v", "15", props)
	}

	if !hasValue(props, "Yes") {
		t.Fatalf("expected a property value of %q, got %#v", "Yes", props)
	}

	if !hasValue(props, "MM") {
		t.Fatalf("expected a property value of %q, got %#v", "MM", props)
	}

	if !hasMultiValueSet(props, []string{"Fire", "Cold", "2"}) {
		t.Fatalf("expected multi-value property containing Fire/Cold/2, got %#v", props)
	}

	if hasLabel(props, "Bad List") {
		t.Fatal("extractProps should skip unsupported list values")
	}
}

func hasLabel(props []Property, label string) bool {
	for _, p := range props {
		if p.Label == label {
			return true
		}
	}
	return false
}

func hasValue(props []Property, want string) bool {
	for _, p := range props {
		for _, v := range p.Values {
			if v == want {
				return true
			}
		}
	}
	return false
}

func hasMultiValueSet(props []Property, values []string) bool {
	want := map[string]bool{}
	for _, v := range values {
		want[v] = true
	}

	for _, p := range props {
		if !p.IsMulti || len(p.Values) != len(values) {
			continue
		}
		seen := map[string]bool{}
		for _, v := range p.Values {
			seen[v] = true
		}
		ok := true
		for v := range want {
			if !seen[v] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
