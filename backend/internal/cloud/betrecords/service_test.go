package betrecords

import (
	"context"
	"testing"
)

func TestGroupsReal(t *testing.T) {
	s := NewService(nil)
	got := s.Groups(context.Background(), 0, ModeReal, 3)
	if got.Mode != ModeReal || got.Days != 3 {
		t.Fatalf("unexpected meta: %+v", got)
	}
	if len(got.Groups.Items) != 5 {
		t.Fatalf("want 5 groups, got %d", len(got.Groups.Items))
	}
	if got.Summary.TotalBet <= 0 {
		t.Fatalf("expected positive totalBet, got %v", got.Summary.TotalBet)
	}
}

func TestGroupsSimEmpty(t *testing.T) {
	s := NewService(nil)
	got := s.Groups(context.Background(), 0, ModeSim, 3)
	if len(got.Groups.Items) != 0 {
		t.Fatalf("sim should be empty")
	}
}

func TestGroupsPagination(t *testing.T) {
	s := NewService(nil)
	first, err := s.GroupsWithFilter(context.Background(), 0, GroupsFilter{
		Mode:  string(ModeReal),
		Days:  3,
		Limit: 2,
	})
	if err != nil || len(first.Groups.Items) != 2 || !first.Groups.Page.HasMore {
		t.Fatalf("first page: err=%v items=%d hasMore=%v", err, len(first.Groups.Items), first.Groups.Page.HasMore)
	}
	second, err := s.GroupsWithFilter(context.Background(), 0, GroupsFilter{
		Mode:   string(ModeReal),
		Days:   3,
		Limit:  2,
		Cursor: *first.Groups.Page.NextCursor,
	})
	if err != nil || len(second.Groups.Items) != 2 || !second.Groups.Page.HasMore {
		t.Fatalf("second page: err=%v items=%d hasMore=%v", err, len(second.Groups.Items), second.Groups.Page.HasMore)
	}
	third, err := s.GroupsWithFilter(context.Background(), 0, GroupsFilter{
		Mode:   string(ModeReal),
		Days:   3,
		Limit:  2,
		Cursor: *second.Groups.Page.NextCursor,
	})
	if err != nil || len(third.Groups.Items) != 1 {
		t.Fatalf("third page: err=%v items=%d", err, len(third.Groups.Items))
	}
}

func TestDetailNotFound(t *testing.T) {
	s := NewService(nil)
	_, ok, err := s.Detail(context.Background(), 0, "missing", ModeReal, 3, 50, "")
	if err != nil || ok {
		t.Fatalf("expected not found")
	}
}

func TestDetailPagination(t *testing.T) {
	s := NewService(nil)
	first, ok, err := s.Detail(context.Background(), 0, "sch-wan", ModeReal, 3, 2, "")
	if err != nil || !ok || len(first.Records.Items) != 2 {
		t.Fatalf("first page: ok=%v err=%v items=%d", ok, err, len(first.Records.Items))
	}
	if !first.Records.Page.HasMore || first.Records.Page.NextCursor == nil {
		t.Fatal("expected hasMore on first page")
	}
	second, ok, err := s.Detail(context.Background(), 0, "sch-wan", ModeReal, 3, 2, *first.Records.Page.NextCursor)
	if err != nil || !ok {
		t.Fatal("second page failed")
	}
	if len(first.Records.Items)+len(second.Records.Items) != 3 {
		t.Fatalf("want 3 total rows for sch-wan")
	}
}
