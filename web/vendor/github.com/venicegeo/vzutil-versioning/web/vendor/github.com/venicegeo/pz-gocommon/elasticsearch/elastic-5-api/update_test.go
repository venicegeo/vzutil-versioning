// Copyright 2012-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestUpdateViaDoc(t *testing.T) {
	client := setupTestClient(t)
	update := client.Update().
		Index("test").Type("type1").Id("1").
		Doc(map[string]interface{}{"name": "new_name"}).
		DetectNoop(true)
	path, params, err := update.url()
	if err != nil {
		t.Fatalf("expected to return URL, got: %v", err)
	}
	expectedPath := `/test/type1/1/_update`
	if expectedPath != path {
		t.Errorf("expected URL path\n%s\ngot:\n%s", expectedPath, path)
	}
	expectedParams := url.Values{}
	if expectedParams.Encode() != params.Encode() {
		t.Errorf("expected URL parameters\n%s\ngot:\n%s", expectedParams.Encode(), params.Encode())
	}
	body, err := update.body()
	if err != nil {
		t.Fatalf("expected to return body, got: %v", err)
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("expected to marshal body as JSON, got: %v", err)
	}
	got := string(data)
	expected := `{"detect_noop":true,"doc":{"name":"new_name"}}`
	if got != expected {
		t.Errorf("expected\n%s\ngot:\n%s", expected, got)
	}
}

func TestUpdateViaDocAndUpsert(t *testing.T) {
	client := setupTestClient(t)
	update := client.Update().
		Index("test").Type("type1").Id("1").
		Doc(map[string]interface{}{"name": "new_name"}).
		DocAsUpsert(true).
		Timeout("1s").
		Refresh("true")
	path, params, err := update.url()
	if err != nil {
		t.Fatalf("expected to return URL, got: %v", err)
	}
	expectedPath := `/test/type1/1/_update`
	if expectedPath != path {
		t.Errorf("expected URL path\n%s\ngot:\n%s", expectedPath, path)
	}
	expectedParams := url.Values{"refresh": []string{"true"}, "timeout": []string{"1s"}}
	if expectedParams.Encode() != params.Encode() {
		t.Errorf("expected URL parameters\n%s\ngot:\n%s", expectedParams.Encode(), params.Encode())
	}
	body, err := update.body()
	if err != nil {
		t.Fatalf("expected to return body, got: %v", err)
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("expected to marshal body as JSON, got: %v", err)
	}
	got := string(data)
	expected := `{"doc":{"name":"new_name"},"doc_as_upsert":true}`
	if got != expected {
		t.Errorf("expected\n%s\ngot:\n%s", expected, got)
	}
}
