// Copyright 2012-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"encoding/json"
	"testing"
)

func TestSortInfo(t *testing.T) {
	builder := SortInfo{Field: "grade", Ascending: false}
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"grade":{"order":"desc"}}`
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}

func TestSortInfoComplex(t *testing.T) {
	builder := SortInfo{
		Field:        "price",
		Ascending:    false,
		Missing:      "_last",
		SortMode:     "avg",
		NestedFilter: NewTermQuery("product.color", "blue"),
		NestedPath:   "variant",
	}
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"price":{"missing":"_last","mode":"avg","nested_filter":{"term":{"product.color":"blue"}},"nested_path":"variant","order":"desc"}}`
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}

func TestScoreSort(t *testing.T) {
	builder := NewScoreSort()
	if builder.ascending != false {
		t.Error("expected score sorter to be ascending by default")
	}
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"_score":{"order":"desc"}}` // ScoreSort is "desc" by default
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}

func TestScoreSortOrderAscending(t *testing.T) {
	builder := NewScoreSort().Asc()
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"_score":{"order":"asc"}}`
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}

func TestScoreSortOrderDescending(t *testing.T) {
	builder := NewScoreSort().Desc()
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"_score":{"order":"desc"}}`
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}

func TestFieldSort(t *testing.T) {
	builder := NewFieldSort("grade")
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"grade":{"order":"asc"}}`
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}

func TestFieldSortOrderDesc(t *testing.T) {
	builder := NewFieldSort("grade").Desc()
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"grade":{"order":"desc"}}`
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}

func TestFieldSortComplex(t *testing.T) {
	builder := NewFieldSort("price").Desc().
		SortMode("avg").
		Missing("_last").
		UnmappedType("product").
		NestedFilter(NewTermQuery("product.color", "blue")).
		NestedPath("variant")
	src, err := builder.Source()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshaling to JSON failed: %v", err)
	}
	got := string(data)
	expected := `{"price":{"missing":"_last","mode":"avg","nested_filter":{"term":{"product.color":"blue"}},"nested_path":"variant","order":"desc","unmapped_type":"product"}}`
	if got != expected {
		t.Errorf("expected\n%s\n,got:\n%s", expected, got)
	}
}
