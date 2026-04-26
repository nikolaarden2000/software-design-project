package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func newRoomRepo(mock pgxmock.PgxPoolIface) *RoomRepo {
	return NewRoomRepo(mock)
}

var roomCols = []string{"id", "title", "company", "address", "capacity", "image_url", "price"}

var roomPageCols = []string{
	"id", "title", "company", "address",
	"images", "price", "capacity",
	"available_from", "available_to",
	"description", "latitude", "longitude",
}

// GetRoomsBatchByCity

func TestGetRoomsBatchByCity_Success(t *testing.T) {
	cases := []struct {
		name      string
		lastID    int
		wantCount int
		wantFirst int
	}{
		{
			name:      "first page (lastID=0)",
			lastID:    0,
			wantCount: 2,
			wantFirst: 1,
		},
		{
			name:      "with cursor (lastID=5)",
			lastID:    5,
			wantCount: 1,
			wantFirst: 6,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			rows := pgxmock.NewRows(roomCols)
			if tc.lastID == 0 {
				rows.
					AddRow(1, "Room A", "Acme", "Moscow, Lenin St, 1", 8, "http://img/a.jpg", 1000).
					AddRow(2, "Room B", "Acme", "Moscow, Lenin St, 2", 12, "http://img/b.jpg", 1500)
			} else {
				rows.AddRow(6, "Room C", "Acme", "Moscow, Lenin St, 3", 6, "http://img/c.jpg", 800)
			}

			mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomsBatchByCity)).
				WithArgs(tc.lastID, "Moscow", 10).
				WillReturnRows(rows)

			rooms, err := newRoomRepo(mock).GetRoomsBatchByCity(context.Background(), tc.lastID, 10, "Moscow")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(rooms) != tc.wantCount {
				t.Fatalf("expected %d rooms, got %d", tc.wantCount, len(rooms))
			}
			if rooms[0].ID != tc.wantFirst {
				t.Errorf("first room ID: got %d, want %d", rooms[0].ID, tc.wantFirst)
			}
		})
	}
}

func TestGetRoomsBatchByCity_UnknownCity_ReturnsEmptySlice(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomsBatchByCity)).
		WithArgs(0, "Samara", 10).
		WillReturnRows(pgxmock.NewRows(roomCols))

	rooms, err := newRoomRepo(mock).GetRoomsBatchByCity(context.Background(), 0, 10, "Samara")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rooms == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(rooms) != 0 {
		t.Errorf("expected 0 rooms, got %d", len(rooms))
	}
}

func TestGetRoomsBatchByCity_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection reset")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomsBatchByCity)).
		WithArgs(0, "Moscow", 10).
		WillReturnError(dbErr)

	_, err := newRoomRepo(mock).GetRoomsBatchByCity(context.Background(), 0, 10, "Moscow")

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestGetRoomsBatchByCity_RowsIterationError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	iterErr := fmt.Errorf("network interrupted")
	rows := pgxmock.NewRows(roomCols).
		AddRow(1, "Room A", "Acme", "Moscow, St, 1", 8, "img", 1000).
		RowError(0, iterErr)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomsBatchByCity)).
		WithArgs(0, "Moscow", 10).
		WillReturnRows(rows)

	_, err := newRoomRepo(mock).GetRoomsBatchByCity(context.Background(), 0, 10, "Moscow")

	if !errors.Is(err, iterErr) {
		t.Fatalf("expected iterErr, got: %v", err)
	}
}
func TestGetRoomsBatchByCity_NegativeLastID_ReturnsErrInvalidID(t *testing.T) {
	mock := newMock(t)

	_, err := newRoomRepo(mock).GetRoomsBatchByCity(context.Background(), -1, 10, "Moscow")

	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got: %v", err)
	}
}

func TestGetRoomsBatchByCity_InvalidLimit_ReturnsErrInvalidArgument(t *testing.T) {
	cases := []struct {
		name  string
		limit int
	}{
		{"zero limit", 0},
		{"negative limit", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			_, err := newRoomRepo(mock).GetRoomsBatchByCity(context.Background(), 0, tc.limit, "Moscow")

			if !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("[%s] expected ErrInvalidArgument, got: %v", tc.name, err)
			}
		})
	}
}

// GetCompaniesByCity

func TestGetCompaniesByCity_ReturnsCompanies(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).
		WithArgs("Moscow").
		WillReturnRows(pgxmock.NewRows([]string{"name"}).
			AddRow("Acme").
			AddRow("Globex"))

	companies, err := newRoomRepo(mock).GetCompaniesByCity(context.Background(), "Moscow")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(companies) != 2 || companies[0] != "Acme" || companies[1] != "Globex" {
		t.Errorf("unexpected companies: %v", companies)
	}
}

func TestGetCompaniesByCity_UnknownCity_ReturnsEmptySlice(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).
		WithArgs("Samara").
		WillReturnRows(pgxmock.NewRows([]string{"name"}))

	companies, err := newRoomRepo(mock).GetCompaniesByCity(context.Background(), "Samara")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if companies == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(companies) != 0 {
		t.Errorf("expected 0 companies, got %d", len(companies))
	}
}

func TestGetCompaniesByCity_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("timeout")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).
		WithArgs("Moscow").
		WillReturnError(dbErr)

	_, err := newRoomRepo(mock).GetCompaniesByCity(context.Background(), "Moscow")

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestGetCompaniesByCity_RowsIterationError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	iterErr := fmt.Errorf("network interrupted")
	rows := pgxmock.NewRows([]string{"name"}).
		AddRow("Acme").
		RowError(0, iterErr)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).
		WithArgs("Moscow").
		WillReturnRows(rows)

	_, err := newRoomRepo(mock).GetCompaniesByCity(context.Background(), "Moscow")

	if !errors.Is(err, iterErr) {
		t.Fatalf("expected iterErr, got: %v", err)
	}
}

// GetRoomPageData

func TestGetRoomPageData_Success(t *testing.T) {
	mock := newMock(t)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomPageData)).
		WithArgs(123).
		WillReturnRows(pgxmock.NewRows(roomPageCols).AddRow(
			123, "Meeting Room", "Acme", "Moscow, Lenin St, 1",
			[]string{"img1.jpg", "img2.jpg"}, 2000, 10,
			availFrom, availTo,
			"A nice room", 48.7423, 44.5370,
		))

	d, err := newRoomRepo(mock).GetRoomPageData(context.Background(), 123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ID != 123 || d.Title != "Meeting Room" {
		t.Errorf("unexpected fields: %+v", d)
	}
	if d.AvailableFrom != "09:00" {
		t.Errorf("AvailableFrom: got %q, want %q", d.AvailableFrom, "09:00")
	}
	if d.AvailableTo != "18:00" {
		t.Errorf("AvailableTo: got %q, want %q", d.AvailableTo, "18:00")
	}
	if d.MaxCapacity != d.Capacity {
		t.Errorf("MaxCapacity (%d) != Capacity (%d)", d.MaxCapacity, d.Capacity)
	}
}

func TestGetRoomPageData_NotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomPageData)).
		WithArgs(999).
		WillReturnRows(pgxmock.NewRows(roomPageCols))

	_, err := newRoomRepo(mock).GetRoomPageData(context.Background(), 999)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetRoomPageData_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection refused")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomPageData)).
		WithArgs(42).
		WillReturnError(dbErr)

	_, err := newRoomRepo(mock).GetRoomPageData(context.Background(), 42)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped dbErr, got: %v", err)
	}
}

func TestGetRoomPageData_InvalidRoomID_ReturnsErrInvalidID(t *testing.T) {
	cases := []struct {
		name   string
		roomID int
	}{
		{"zero id", 0},
		{"negative id", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			_, err := newRoomRepo(mock).GetRoomPageData(context.Background(), tc.roomID)

			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("[%s] expected ErrInvalidID, got: %v", tc.name, err)
			}
		})
	}
}
