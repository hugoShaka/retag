package main

import (
	"testing"
)

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

const testToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiIsImtpZCI6IkJWM0Q6MkFWWjpVQjVaOktJQVA6SU5QTDo1RU42Ok40SjQ6Nk1XTzpEUktFOkJWUUs6M0ZKTDpQT1RMIn0.eyJpc3MiOiJhdXRoLmRvY2tlci5jb20iLCJzdWIiOiJCQ0NZOk9VNlo6UUVKNTpXTjJDOjJBVkM6WTdZRDpBM0xZOjQ1VVc6NE9HRDpLQUxMOkNOSjU6NUlVTCIsImF1ZCI6InJlZ2lzdHJ5LmRvY2tlci5jb20iLCJleHAiOjE0MTUzODczMTUsIm5iZiI6MTQxNTM4NzAxNSwiaWF0IjoxNDE1Mzg3MDE1LCJqdGkiOiJ0WUpDTzFjNmNueXk3a0FuMGM3cktQZ2JWMUgxYkZ3cyIsInNjb3BlIjoiamxoYXduOnJlcG9zaXRvcnk6c2FtYWxiYS9teS1hcHA6cHVzaCxwdWxsIGpsaGF3bjpuYW1lc3BhY2U6c2FtYWxiYTpwdWxsIn0.Y3zZSwaZPqy4y9oRBVRImZyv3m_S9XDHF1tWwN7mL52C_IiA73SJkWVNsvNqpJIn5h7A2F8biv_S2ppQ1lgkbw"
const testHeader = "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiIsImtpZCI6IkJWM0Q6MkFWWjpVQjVaOktJQVA6SU5QTDo1RU42Ok40SjQ6Nk1XTzpEUktFOkJWUUs6M0ZKTDpQT1RMIn0.eyJpc3MiOiJhdXRoLmRvY2tlci5jb20iLCJzdWIiOiJCQ0NZOk9VNlo6UUVKNTpXTjJDOjJBVkM6WTdZRDpBM0xZOjQ1VVc6NE9HRDpLQUxMOkNOSjU6NUlVTCIsImF1ZCI6InJlZ2lzdHJ5LmRvY2tlci5jb20iLCJleHAiOjE0MTUzODczMTUsIm5iZiI6MTQxNTM4NzAxNSwiaWF0IjoxNDE1Mzg3MDE1LCJqdGkiOiJ0WUpDTzFjNmNueXk3a0FuMGM3cktQZ2JWMUgxYkZ3cyIsInNjb3BlIjoiamxoYXduOnJlcG9zaXRvcnk6c2FtYWxiYS9teS1hcHA6cHVzaCxwdWxsIGpsaGF3bjpuYW1lc3BhY2U6c2FtYWxiYTpwdWxsIn0.Y3zZSwaZPqy4y9oRBVRImZyv3m_S9XDHF1tWwN7mL52C_IiA73SJkWVNsvNqpJIn5h7A2F8biv_S2ppQ1lgkbw"

func TestCreateRequest(t *testing.T) {
	tables := []struct {
		verb  string
		url   string
		token string
	}{
		{"GET", "http://reg.local", testToken},
		{"POST", "https://reg.local", ""},
		{"GET", "https://toto", testToken},
	}
	for _, table := range tables {
		req := createRequest(table.verb, table.url, nil, table.token)
		header := req.Header.Get("Authorization")
		switch {
		case len(table.token) == 0 && len(header) != 0:
			t.Errorf("Authorization header should not be set, got %s", header)
		case len(table.token) != 0 && header != testHeader:
			t.Errorf("Invalid header se to request, got :%s, want: %s", header, testHeader)
		default:
		}
	}
}
