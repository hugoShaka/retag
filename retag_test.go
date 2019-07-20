package main

import "testing"

func TestParseImageRef(t *testing.T) {

	tables := []struct {
		ref        string
		registry   string
		repository string
		tag        string
		fail       bool
	}{
		{"reg.local/image1:v1", "reg.local", "image1", "v1", false},
		{"reg.local/image2", "reg.local", "image2", "latest", false},
		{"reg.local", "", "", "", true},
		{"reg.local/foo/bar/image:v12", "reg.local", "foo/bar/image", "v12", false},
		{"reg.local/foo/bar/image:v13:invalid", "", "", "", true},
	}
	for _, table := range tables {
		parsedRef, err := parseImageRef(table.ref)
		failed := (err != nil)
		if table.fail != failed {
			t.Errorf("Parsing of (%s) went wrong, failed: %t, should fail : %t", table.ref, failed, table.fail)
		} else {
			if parsedRef.registry != table.registry {
				t.Errorf("Parsing registry of (%s) was incorrect, got: %s, want: %s.", table.ref, parsedRef.registry, table.registry)
			}
			if parsedRef.repository != table.repository {
				t.Errorf("Parsing repository of (%s) was incorrect, got: %s, want: %s.", table.ref, parsedRef.repository, table.repository)
			}
			if parsedRef.tag != table.tag {
				t.Errorf("Parsing tag of (%s) was incorrect, got: %s, want: %s.", table.ref, parsedRef.tag, table.tag)
			}
		}

	}

}
