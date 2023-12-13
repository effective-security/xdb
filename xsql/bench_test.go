package xsql_test

import (
	"fmt"
	"testing"

	"github.com/effective-security/xdb/xsql"
)

var s string

func BenchmarkSelectDontClose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := xsql.Select("id").From("table").Where("id > ?", 42).Where("id < ?", 1000)
		s = q.String()
	}
}

func BenchmarkSelect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := xsql.Select("id").From("table").Where("id > ?", 42).Where("id < ?", 1000)
		s = q.String()
		q.Close()
	}
}

func BenchmarkSelectPg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := xsql.Postgres.Select("id").From("table").Where("id > ?", 42).Where("id < ?", 1000)
		s = q.String()
		q.Close()
	}
}

func BenchmarkManyFields(b *testing.B) {
	fields := make([]string, 0, 100)

	for n := 1; n <= cap(fields); n++ {
		fields = append(fields, fmt.Sprintf("field_%d", n))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q := xsql.Select("id").From("table").Where("id > ?", 42).Where("id < ?", 1000)
		for _, field := range fields {
			q.Select(field)
		}
		s = q.String()
		q.Close()
	}
}

func BenchmarkBind(b *testing.B) {
	type Record struct {
		ID int64 `db:"id"`
	}
	var u struct {
		Record
		Name string `db:"name"`
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q := xsql.From("table").Bind(&u).Where("id = ?", 42)
		s = q.String()
		q.Close()
	}
}

func BenchmarkManyFieldsPg(b *testing.B) {
	fields := make([]string, 0, 100)

	for n := 1; n <= cap(fields); n++ {
		fields = append(fields, fmt.Sprintf("field_%d", n))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q := xsql.Postgres.Select("id").From("table").Where("id > ?", 42).Where("id < ?", 1000)
		for _, field := range fields {
			q.Select(field)
		}
		s = q.String()
		q.Close()
	}
}

func BenchmarkMixedOrder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := xsql.Select("id").Where("id > ?", 42).From("table").Where("id < ?", 1000)
		s = q.String()
		q.Close()
	}
}

func BenchmarkBuildPg(b *testing.B) {
	q := xsql.Postgres.Select("id").From("table").Where("id > ?", 42).Where("id < ?", 1000)

	for i := 0; i < b.N; i++ {
		q.Invalidate()
		s = q.String()
	}
}

func BenchmarkBuild(b *testing.B) {
	q := xsql.Select("id").From("table").Where("id > ?", 42).Where("id < ?", 1000)

	for i := 0; i < b.N; i++ {
		q.Invalidate()
		s = q.String()
	}
}

func BenchmarkDest(b *testing.B) {
	var (
		field1 int
		field2 string
	)
	for i := 0; i < b.N; i++ {
		q := xsql.From("table").
			Select("field1").To(&field1).
			Select("field2").To(&field2)
		q.Close()
	}
}

func selectComplex(b *testing.B, dialect xsql.SQLDialect) {
	for n := 0; n < b.N; n++ {
		q := dialect.Select("DISTINCT a, b, z, y, x").
			From("c").
			Where("(d = ? OR e = ?)", 1, "wat").
			Where("f = ? and x = ?", 2, "hi").
			Where("g = ?", 3).
			Where("h").In(1, 2, 3).
			GroupBy("i").
			GroupBy("ii").
			GroupBy("iii").
			Having("j = k").
			Having("jj = ?", 1).
			Having("jjj = ?", 2).
			OrderBy("l").
			OrderBy("l").
			OrderBy("l").
			Limit(7).
			Offset(8)
		s = q.String()
		q.Close()
	}
}

func selectSubqueryFmt(b *testing.B, dialect xsql.SQLDialect) {
	for n := 0; n < b.N; n++ {
		sq := dialect.Select("id").
			From("tickets").
			Where("subdomain_id = ? and (state = ? or state = ?)", 1, "open", "spam")
		subQuery := sq.String()

		q := dialect.Select("DISTINCT a, b").
			Select(fmt.Sprintf("(%s) AS subq", subQuery)).
			From("c").
			Where("f = ? and x = ?", 2, "hi").
			Where("g = ?", 3).
			OrderBy("l").
			OrderBy("l").
			Limit(7).
			Offset(8)
		s = q.String()
		q.Close()
		sq.Close()
	}
}

func selectSubquery(b *testing.B, dialect xsql.SQLDialect) {
	for n := 0; n < b.N; n++ {
		q := dialect.Select("DISTINCT a, b").
			SubQuery("(", ") AS subq", xsql.Select("id").
				From("tickets").
				Where("subdomain_id = ? and (state = ? or state = ?)", 1, "open", "spam")).
			From("c").
			Where("f = ? and x = ?", 2, "hi").
			Where("g = ?", 3).
			OrderBy("l").
			OrderBy("l").
			Limit(7).
			Offset(8)
		s = q.String()
		q.Close()
	}
}

func BenchmarkSelectComplex(b *testing.B) {
	selectComplex(b, xsql.NoDialect)
}

func BenchmarkSelectComplexPg(b *testing.B) {
	selectComplex(b, xsql.Postgres)
}

func BenchmarkSelectSubqueryFmt(b *testing.B) {
	selectSubqueryFmt(b, xsql.NoDialect)
}

func BenchmarkSelectSubqueryFmtPostgreSQL(b *testing.B) {
	selectSubqueryFmt(b, xsql.Postgres)
}

func BenchmarkSelectSubquery(b *testing.B) {
	selectSubquery(b, xsql.NoDialect)
}

func BenchmarkSelectSubqueryPostgreSQL(b *testing.B) {
	selectSubquery(b, xsql.Postgres)
}

func BenchmarkWith(b *testing.B) {
	for n := 0; n < b.N; n++ {
		q := xsql.From("orders").
			With("regional_sales",
				xsql.From("orders").
					Select("region, SUM(amount) AS total_sales").
					GroupBy("region")).
			With("top_regions",
				xsql.From("regional_sales").
					Select("region").
					Where("total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)")).
			Select("region").
			Select("product").
			Select("SUM(quantity) AS product_units").
			Select("SUM(amount) AS product_sales").
			Where("region IN (SELECT region FROM top_regions)").
			GroupBy("region, product")
		s = q.String()
		q.Close()
	}
}

func BenchmarkIn(b *testing.B) {
	a := make([]any, 50)
	for i := 0; i < len(a); i++ {
		a[i] = i + 1
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		q := xsql.From("orders").
			Select("id").
			Where("status").In(a...)
		s = q.String()
		q.Close()
	}
}
