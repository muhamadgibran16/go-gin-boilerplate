package pagination

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want Query
	}{
		{Query{Page: 2, PerPage: 20}, Query{Page: 2, PerPage: 20}},
		{Query{}, Query{Page: 1, PerPage: 10}},
		{Query{Page: -1, PerPage: -5}, Query{Page: 1, PerPage: 10}},
		{Query{Page: 1, PerPage: 1000}, Query{Page: 1, PerPage: 1000}}, // the 100 limit is enforced by binding, not here
	}
	for _, tt := range tests {
		if got := tt.in.Normalize(); got != tt.want {
			t.Errorf("%+v.Normalize() = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestOffsetAndLimit(t *testing.T) {
	tests := []struct {
		q                     Query
		wantOffset, wantLimit int
	}{
		{Query{Page: 1, PerPage: 10}, 0, 10},
		{Query{Page: 2, PerPage: 10}, 10, 10},
		{Query{Page: 3, PerPage: 25}, 50, 25},
		{Query{}, 0, 10}, // zero value falls back to defaults
	}
	for _, tt := range tests {
		if got := tt.q.Offset(); got != tt.wantOffset {
			t.Errorf("%+v.Offset() = %d, want %d", tt.q, got, tt.wantOffset)
		}
		if got := tt.q.Limit(); got != tt.wantLimit {
			t.Errorf("%+v.Limit() = %d, want %d", tt.q, got, tt.wantLimit)
		}
	}
}

func TestTotalPages(t *testing.T) {
	tests := []struct {
		total   int64
		perPage int
		want    int
	}{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{5, 0, 0},
	}
	for _, tt := range tests {
		if got := TotalPages(tt.total, tt.perPage); got != tt.want {
			t.Errorf("TotalPages(%d, %d) = %d, want %d", tt.total, tt.perPage, got, tt.want)
		}
	}
}

func TestNewMeta(t *testing.T) {
	got := NewMeta(Query{Page: 2, PerPage: 10}, 3, 13)
	want := Meta{CurrentPage: 2, PerPage: 10, TotalCurrentPage: 3, TotalPage: 2, TotalData: 13}
	if got != want {
		t.Fatalf("NewMeta() = %+v, want %+v", got, want)
	}
}
