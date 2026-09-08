package xsql_test

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/effective-security/xdb/xsql"
)

type dummyDB int

func (db *dummyDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (db *dummyDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}

func (db *dummyDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

var db = new(dummyDB)
var ctx = context.Background()

func Example() {
	var (
		region       string
		product      string
		productUnits int
		productSales float64
	)

	pg := xsql.Postgres.WithNewLines(false)
	err := pg.From("orders").
		With("regional_sales",
			pg.From("orders").
				Select("region, SUM(amount) AS total_sales").
				GroupBy("region")).
		With("top_regions",
			pg.From("regional_sales").
				Select("region").
				Where("total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)")).
		// Map query fields to variables
		Select("region").To(&region).
		Select("product").To(&product).
		Select("SUM(quantity)").To(&productUnits).
		Select("SUM(amount) AS product_sales").To(&productSales).
		//
		Where("region IN (SELECT region FROM top_regions)").
		GroupBy("region, product").
		OrderBy("product_sales DESC").
		// Execute the query
		QueryAndClose(ctx, db, func(row *sql.Rows) {
			// Callback function is called for every returned row.
			// Row values are scanned automatically to bound variables.
			fmt.Printf("%s\t%s\t%d\t$%.2f\n", region, product, productUnits, productSales)
		})
	if err != nil {
		panic(err)
	}
}

func ExampleStmt_OrderBy() {
	q := xsql.UseNewLines(false).Select("id").From("table").OrderBy("id", "name DESC")
	fmt.Println(q.String())
	// Output: SELECT id FROM table ORDER BY id, name DESC
}

func ExampleStmt_Limit() {
	q := xsql.UseNewLines(false).Select("id").From("table").Limit(10)
	fmt.Println(q.String())
	// Output: SELECT id FROM table LIMIT ?
}

func ExampleStmt_Offset() {
	q := xsql.UseNewLines(false).Select("id").From("table").Limit(10).Offset(10)
	fmt.Println(q.String())
	// Output: SELECT id FROM table LIMIT ? OFFSET ?
}

func ExampleStmt_Paginate() {
	q := xsql.UseNewLines(false).Select("id").From("table").Paginate(5, 10)
	fmt.Println(q.String(), q.Args())
	q.Close()

	q = xsql.UseNewLines(false).Select("id").From("table").Paginate(1, 10)
	fmt.Println(q.String(), q.Args())
	q.Close()

	// Zero and negative values are replaced with 1
	q = xsql.UseNewLines(false).Select("id").From("table").Paginate(-1, -1)
	fmt.Println(q.String(), q.Args())
	q.Close()

	// Output:
	// SELECT id FROM table LIMIT ? OFFSET ? [10 40]
	// SELECT id FROM table LIMIT ? [10]
	// SELECT id FROM table LIMIT ? [1]
}

func ExampleStmt_Update() {
	q := xsql.UseNewLines(false).Update("table").Set("field1", "newvalue").Where("id = ?", 42)
	fmt.Println(q.String(), q.Args())
	q.Close()
	// Output:
	// UPDATE table SET field1=? WHERE id = ? [newvalue 42]
}

func ExampleStmt_SetExpr() {
	q := xsql.UseNewLines(false).Update("table").SetExpr("field1", "field2 + 1").Where("id = ?", 42)
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// UPDATE table SET field1=field2 + 1 WHERE id = ?
	// [42]
}

func ExampleStmt_InsertInto() {
	q := xsql.UseNewLines(false).InsertInto("table").
		Set("field1", "newvalue").
		SetExpr("field2", "field2 + 1")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// INSERT INTO table ( field1, field2 ) VALUES ( ?, field2 + 1 )
	// [newvalue]
}

func ExampleStmt_DeleteFrom() {
	q := xsql.UseNewLines(false).DeleteFrom("table").Where("id = ?", 42)
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// DELETE FROM table WHERE id = ?
	// [42]
}

func ExampleStmt_GroupBy() {
	q := xsql.UseNewLines(false).From("incomes").
		Select("source, sum(amount) as s").
		Where("amount > ?", 42).
		GroupBy("source")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// SELECT source, sum(amount) as s FROM incomes WHERE amount > ? GROUP BY source
	// [42]
}

func ExampleStmt_Having() {
	q := xsql.UseNewLines(false).From("incomes").
		Select("source, sum(amount) as s").
		Where("amount > ?", 42).
		GroupBy("source").
		Having("s > ?", 100)
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// SELECT source, sum(amount) as s FROM incomes WHERE amount > ? GROUP BY source HAVING s > ?
	// [42 100]
}

func ExampleStmt_Returning() {
	var newId int
	q := xsql.UseNewLines(false).InsertInto("table").
		Set("field1", "newvalue").
		Returning("id").To(&newId)
	fmt.Println(q.String(), q.Args())
	q.Close()
	// Output:
	// INSERT INTO table ( field1 ) VALUES ( ? ) RETURNING id [newvalue]
}

func ExamplePostgres() {
	q := xsql.Postgres.WithNewLines(false).From("table").Select("field").Where("id = ?", 42)
	fmt.Println(q.String())
	q.Close()
	// Output:
	// SELECT field FROM table WHERE id = $1
}

func ExampleStmt_OnConflict() {
	q := xsql.Postgres.WithNewLines(false).
		InsertInto("users").
		Set("email", "a@b.c").
		Set("name", "Ada").
		OnConflict("(email) DO UPDATE SET name = EXCLUDED.name").
		Returning("id")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// INSERT INTO users ( email, name ) VALUES ( $1, $2 ) ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name RETURNING id
	// [a@b.c Ada]
}

func ExampleStmt_ForUpdate() {
	q := xsql.Postgres.WithNewLines(false).
		Select("id, balance").
		From("accounts").
		Where("id = ?", 42).
		ForUpdate("NOWAIT")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// SELECT id, balance FROM accounts WHERE id = $1 FOR UPDATE NOWAIT
	// [42]
}

func ExampleStmt_NotIn() {
	q := xsql.UseNewLines(false).From("tasks").
		Select("id, status").
		Where("status").NotIn("done", "canceled")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// SELECT id, status FROM tasks WHERE status NOT IN (?,?)
	// [done canceled]
}

func ExampleStmt_NewRow() {
	q := xsql.Postgres.WithNewLines(false).InsertInto("users")
	q.NewRow().Set("email", "first@ex.com").Set("name", "First")
	q.NewRow().Set("email", "second@ex.com").Set("name", "Second")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// INSERT INTO users ( email, name ) VALUES ( $1, $2 ), ( $3, $4 )
	// [first@ex.com First second@ex.com Second]
}

func ExampleStmt_Clone() {
	base := xsql.UseNewLines(false).From("users").Select("id, name").Where("active = ?", true)
	q := base.Clone().Where("id = ?", 7).Limit(1)
	fmt.Println(q.String())
	fmt.Println(q.Args())
	fmt.Println(base.String())
	q.Close()
	base.Close()
	// Output:
	// SELECT id, name FROM users WHERE active = ? AND id = ? LIMIT ?
	// [true 7 1]
	// SELECT id, name FROM users WHERE active = ?
}

func ExampleStmt_Join() {
	q := xsql.Postgres.WithNewLines(false).
		From("orders o").
		Select("o.id, u.email").
		Join("users u", "u.id = o.user_id").
		Where("o.total > ?", 100)
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()
	// Output:
	// SELECT o.id, u.email FROM orders o JOIN users u ON (u.id = o.user_id) WHERE o.total > $1
	// [100]
}

func ExampleStmt_With() {
	nl := xsql.UseNewLines(false)
	q := nl.From("orders").
		With("regional_sales",
			nl.From("orders").
				Select("region, SUM(amount) AS total_sales").
				GroupBy("region")).
		With("top_regions",
			nl.From("regional_sales").
				Select("region").
				Where("total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)")).
		Select("region").
		Select("product").
		Select("SUM(quantity) AS product_units").
		Select("SUM(amount) AS product_sales").
		Where("region IN (SELECT region FROM top_regions)").
		GroupBy("region, product")
	fmt.Println(q.String())
	q.Close()
	// Output:
	// WITH regional_sales AS (SELECT region, SUM(amount) AS total_sales FROM orders GROUP BY region), top_regions AS (SELECT region FROM regional_sales WHERE total_sales > (SELECT SUM(total_sales)/10 FROM regional_sales)) SELECT region, product, SUM(quantity) AS product_units, SUM(amount) AS product_sales FROM orders WHERE region IN (SELECT region FROM top_regions) GROUP BY region, product
}

func ExampleStmt_From() {
	nl := xsql.UseNewLines(false)
	q := nl.Select("*").
		From("").
		SubQuery(
			"(", ") counted_news",
			nl.From("news").
				Select("id, section, header, score").
				Select("row_number() OVER (PARTITION BY section ORDER BY score DESC) AS rating_in_section").
				OrderBy("section, rating_in_section")).
		Where("rating_in_section <= 5")
	fmt.Println(q.String())
	q.Close()
	// Output:
	//SELECT * FROM (SELECT id, section, header, score, row_number() OVER (PARTITION BY section ORDER BY score DESC) AS rating_in_section FROM news ORDER BY section, rating_in_section) counted_news WHERE rating_in_section <= 5
}

func ExampleStmt_SubQuery() {
	nl := xsql.UseNewLines(false)
	q := nl.From("orders o").
		Select("date, region").
		SubQuery("(", ") AS prev_order_date",
			nl.From("orders po").
				Select("date").
				Where("region = o.region").
				Where("id < o.id").
				OrderBy("id DESC").
				Clause("LIMIT 1")).
		Where("date > CURRENT_DATE - interval '1 day'").
		OrderBy("id DESC")
	fmt.Println(q.String())
	q.Close()

	// Output:
	// SELECT date, region, (SELECT date FROM orders po WHERE region = o.region AND id < o.id ORDER BY id DESC LIMIT 1) AS prev_order_date FROM orders o WHERE date > CURRENT_DATE - interval '1 day' ORDER BY id DESC
}

func ExampleStmt_Clause() {
	q := xsql.UseNewLines(false).From("empsalary").
		Select("sum(salary) OVER w").
		Clause("WINDOW w AS (PARTITION BY depname ORDER BY salary DESC)")
	fmt.Println(q.String())
	q.Close()

	// Output:
	// SELECT sum(salary) OVER w FROM empsalary WINDOW w AS (PARTITION BY depname ORDER BY salary DESC)
}

func ExampleStmt_QueryRowAndClose() {
	type Offer struct {
		id        int64
		productId int64
		price     float64
		isDeleted bool
	}

	var o Offer

	err := xsql.From("offers").
		Select("id").To(&o.id).
		Select("product_id").To(&o.productId).
		Select("price").To(&o.price).
		Select("is_deleted").To(&o.isDeleted).
		Where("id = ?", 42).
		QueryRowAndClose(ctx, db)
	if err != nil {
		panic(err)
	}
}

func ExampleStmt_Bind() {
	type Offer struct {
		Id        int64   `db:"id"`
		ProductId int64   `db:"product_id"`
		Price     float64 `db:"price"`
		IsDeleted bool    `db:"is_deleted"`
	}

	var o Offer

	err := xsql.From("offers").
		Bind(&o).
		Where("id = ?", 42).
		QueryRowAndClose(ctx, db)
	if err != nil {
		panic(err)
	}
}

func ExampleStmt_In() {
	q := xsql.UseNewLines(false).From("tasks").
		Select("id, status").
		Where("status").In("new", "pending", "wip")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()

	// Output:
	// SELECT id, status FROM tasks WHERE status IN (?,?,?)
	// [new pending wip]
}

func ExampleStmt_Union() {
	nl := xsql.UseNewLines(false)
	q := nl.From("tasks").
		Select("id, status").
		Where("status = ?", "new").
		Union(true, nl.From("tasks").
			Select("id, status").
			Where("status = ?", "pending")).
		Union(true, nl.From("tasks").
			Select("id, status").
			Where("status = ?", "wip")).
		OrderBy("id")
	fmt.Println(q.String())
	fmt.Println(q.Args())
	q.Close()

	// Output:
	// SELECT id, status FROM tasks WHERE status = ? UNION ALL SELECT id, status FROM tasks WHERE status = ? UNION ALL SELECT id, status FROM tasks WHERE status = ? ORDER BY id
	// [new pending wip]
}
