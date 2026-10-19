package xdb_test

import (
	"context"
	"database/sql"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/effective-security/xdb"
	"github.com/effective-security/xdb/xsql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// event is a row model as the generator writes it
type event struct {
	ID        xdb.ID
	CreatedAt xdb.Time
	Title     string
}

func (m *event) ScanRow(rows xdb.Row) error {
	return rows.Scan(&m.ID, &m.CreatedAt, &m.Title)
}

type eventResult struct {
	Rows        []*event
	NextOffset  uint32
	HasNextPage bool
	Cursor      string
}

func (p *eventResult) SetResult(rows []*event, hasNextPage bool, nextOffset uint32) {
	p.Rows = rows
	p.NextOffset = nextOffset
	p.HasNextPage = hasNextPage
}

func (p *eventResult) SetResultWithCursor(rows []*event, hasNextPage bool, cursor func(lastRow *event) string) {
	p.Rows = rows
	p.HasNextPage = hasNextPage
	p.Cursor = ""
	if hasNextPage && len(rows) > 0 {
		p.Cursor = cursor(rows[len(rows)-1])
	}
}

// openEvents returns an in-memory database with n events; every three
// events share a created_at, and consecutive ones are 1µs apart.
func openEvents(t *testing.T, n int) (*sql.DB, []*event) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	// one connection: every connection to :memory: is another database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("CREATE TABLE events (id INTEGER PRIMARY KEY, created_at TIMESTAMP NOT NULL, title TEXT NOT NULL)")
	require.NoError(t, err)

	base := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	var all []*event
	for i := range n {
		e := &event{
			ID:        xdb.ID(i + 1),
			CreatedAt: xdb.Time(base.Add(time.Duration(i/3) * time.Microsecond)),
			Title:     "event",
		}
		_, err = db.Exec("INSERT INTO events (id, created_at, title) VALUES (?, ?, ?)", e.ID, e.CreatedAt, e.Title)
		require.NoError(t, err)
		all = append(all, e)
	}
	// ORDER BY created_at DESC, id DESC
	slices.Reverse(all)
	return db, all
}

func TestQueryRows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, all := openEvents(t, 5)

	var ids []xdb.ID
	for m, err := range xdb.QueryRows[event](ctx, db, "SELECT id, created_at, title FROM events ORDER BY created_at DESC, id DESC") {
		require.NoError(t, err)
		ids = append(ids, m.ID)
		if len(ids) == 2 {
			break
		}
	}
	assert.Equal(t, []xdb.ID{all[0].ID, all[1].ID}, ids)

	// break closed the rows: the only connection is free for the next query
	list, err := xdb.ExecuteListQuery[event](ctx, db, "SELECT id, created_at, title FROM events")
	require.NoError(t, err)
	assert.Len(t, list, 5)

	// a query error, and a scan error, end the sequence with the error
	for m, err := range xdb.QueryRows[event](ctx, db, "SELECT nope FROM events") {
		assert.Nil(t, m)
		assert.Error(t, err)
	}
	_, err = xdb.ExecuteListQuery[event](ctx, db, "SELECT id, title FROM events")
	assert.Error(t, err)

	list, err = xdb.ExecuteListQuery[event](ctx, db, "SELECT id, created_at, title FROM events WHERE id < 0")
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Empty(t, list)
}

func TestExecuteQueryWithPaginationLimit(t *testing.T) {
	ctx := context.Background()
	db, _ := openEvents(t, 3)

	// QueryParams without a page: the limit is 0, there is no next page
	var rs eventResult
	err := xdb.ExecuteQueryWithPagination(ctx, db, &rs, "SELECT id, created_at, title FROM events", xdb.NewQueryParams("all"))
	require.NoError(t, err)
	assert.Len(t, rs.Rows, 3)
	assert.False(t, rs.HasNextPage)

	// a full page reports a next page
	qp := xdb.NewQueryParams("page").SetPage(2, 0)
	err = xdb.ExecuteQueryWithPagination(ctx, db, &rs, "SELECT id, created_at, title FROM events ORDER BY id LIMIT ? OFFSET ?", qp)
	require.NoError(t, err)
	assert.Len(t, rs.Rows, 2)
	assert.True(t, rs.HasNextPage)
	assert.Equal(t, uint32(2), rs.NextOffset)

	err = xdb.ExecuteQueryWithPagination(ctx, db, &rs, "SELECT id, created_at, title FROM events ORDER BY id LIMIT ? OFFSET ?", 2, 2)
	require.NoError(t, err)
	assert.Len(t, rs.Rows, 1)
	assert.False(t, rs.HasNextPage)
}

func TestExecuteQueryWithPaginationExactPages(t *testing.T) {
	ctx := context.Background()
	db, _ := openEvents(t, 4)
	const q = "SELECT id, created_at, title FROM events ORDER BY id LIMIT ? OFFSET ?"

	// 4 rows, 2 per page: the second page is the last one
	qp := xdb.NewQueryParams("page").SetPage(2, 0)
	var rs eventResult
	require.NoError(t, xdb.ExecuteQueryWithPagination(ctx, db, &rs, q, qp))
	assert.Equal(t, []xdb.ID{1, 2}, ids(rs.Rows))
	assert.True(t, rs.HasNextPage)
	assert.Equal(t, uint32(2), rs.NextOffset)
	assert.Equal(t, []any{uint32(2), uint32(0)}, qp.Args(), "the params keep the limit")

	args := []any{2, 2}
	require.NoError(t, xdb.ExecuteQueryWithPagination(ctx, db, &rs, q, args...))
	assert.Equal(t, []xdb.ID{3, 4}, ids(rs.Rows))
	assert.False(t, rs.HasNextPage, "no empty page after an exact multiple")
	assert.Zero(t, rs.NextOffset)
	assert.Equal(t, []any{2, 2}, args, "the arguments keep the limit")

	// a named limit
	err := xdb.ExecuteQueryWithPagination(ctx, db, &rs, "SELECT id, created_at, title FROM events ORDER BY id LIMIT :limit OFFSET :offset",
		sql.Named("limit", 3), sql.Named("offset", 0))
	require.NoError(t, err)
	assert.Equal(t, []xdb.ID{1, 2, 3}, ids(rs.Rows))
	assert.True(t, rs.HasNextPage)

	// the largest limit is not increased, it would wrap to LIMIT 0
	qp = xdb.NewQueryParams("page").SetPage(math.MaxUint32, 0)
	require.NoError(t, xdb.ExecuteQueryWithPagination(ctx, db, &rs, q, qp))
	assert.Len(t, rs.Rows, 4)
	assert.False(t, rs.HasNextPage)
}

func ids(rows []*event) []xdb.ID {
	var res []xdb.ID
	for _, e := range rows {
		res = append(res, e.ID)
	}
	return res
}

// listEventsRequest pages events newest first, by the cursor of the
// previous page
type listEventsRequest struct {
	Limit uint32
	// the sort key of the last row of the previous page
	AfterCreatedAt xdb.Time
	AfterID        xdb.ID
}

// setCursor decodes the cursor of the previous page; an empty one starts
// at the first page
func (r *listEventsRequest) setCursor(cursor string) error {
	c, err := xdb.ParseCursor(cursor)
	if err != nil || c.IsZero() {
		return err
	}
	return c.Scan(&r.AfterCreatedAt, &r.AfterID)
}

func (r *listEventsRequest) QueryParams() xdb.QueryParams {
	b := xdb.NewQueryParams("TestKeysetEvents")
	if r.AfterID != 0 {
		b.SetCursor(r.Limit, 2, r.AfterCreatedAt, r.AfterID)
	} else {
		b.SetCursor(r.Limit, 2)
	}
	return b
}

func listEvents(p xdb.QueryParams) string {
	q, _ := xsql.NoDialect.GetOrCreateQuery(p.Name(), func(name string) xsql.Builder {
		q := xsql.NoDialect.From("events").
			Select("id, created_at, title").
			OrderBy("created_at DESC, id DESC")
		if _, cursor := p.Cursor(); cursor != nil {
			q.Where("(created_at, id) < (?, ?)", nil, nil).Limit(nil)
		} else {
			q.Limit(nil).Offset(nil)
		}
		return q
	})
	return q
}

func eventCursor(last *event) string {
	// a Time and an ID always encode
	c, _ := xdb.NewCursor(last.CreatedAt, last.ID)
	return c.String()
}

func TestExecuteQueryWithCursorKeyset(t *testing.T) {
	ctx := context.Background()

	tcases := []struct {
		rows  int
		pages int
	}{
		{rows: 10, pages: 3},
		// an exact multiple of the limit: no empty last page
		{rows: 8, pages: 2},
		{rows: 0, pages: 1},
	}
	for _, tc := range tcases {
		db, all := openEvents(t, tc.rows)

		var seen []xdb.ID
		cursor := ""
		pages := 0
		for {
			require.Less(t, pages, 10, "the listing ends")
			pages++

			req := &listEventsRequest{Limit: 4}
			require.NoError(t, req.setCursor(cursor))
			qp := req.QueryParams()

			var rs eventResult
			require.NoError(t, xdb.ExecuteQueryWithCursor(ctx, db, eventCursor, &rs, listEvents(qp), qp))
			assert.LessOrEqual(t, len(rs.Rows), 4)
			seen = append(seen, ids(rs.Rows)...)
			if !rs.HasNextPage {
				assert.Empty(t, rs.Cursor)
				break
			}
			cursor = rs.Cursor
		}

		assert.Equal(t, ids(all), seen, "every row once, in order, across rows 1µs apart and equal created_at")
		assert.Equal(t, tc.pages, pages, "%d rows", tc.rows)
	}

	req := &listEventsRequest{}
	assert.Error(t, req.setCursor(newCursor(t, xdb.ID(1)).String()), "a cursor of another listing")
}

func TestExecuteQueryWithCursorLimitOnly(t *testing.T) {
	ctx := context.Background()
	db, all := openEvents(t, 4)

	// a lone argument is a filter value, not a limit: the query runs as is
	var rs eventResult
	err := xdb.ExecuteQueryWithCursor(ctx, db, eventCursor, &rs,
		"SELECT id, created_at, title FROM events WHERE id > ? ORDER BY id", 2)
	require.NoError(t, err)
	assert.Equal(t, []xdb.ID{3, 4}, ids(rs.Rows))
	assert.False(t, rs.HasNextPage)

	// the limit of an unfiltered first page is marked by QueryParams
	qp := xdb.NewQueryParams("TestLimitOnly").SetCursor(2, 1)
	err = xdb.ExecuteQueryWithCursor(ctx, db, eventCursor, &rs,
		"SELECT id, created_at, title FROM events ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?", qp)
	require.NoError(t, err)
	assert.Equal(t, ids(all[:2]), ids(rs.Rows))
	assert.True(t, rs.HasNextPage)
	assert.Equal(t, eventCursor(rs.Rows[1]), rs.Cursor)
}
