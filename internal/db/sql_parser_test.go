package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitStatements_DollarQuoting(t *testing.T) {
	sql := `CREATE FUNCTION notify_change() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('changes', NEW.id);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TABLE foo (id int)`

	stmts := splitStatements(sql)

	require.Len(t, stmts, 2)
	require.Contains(t, stmts[0], "notify_change")
	require.Contains(t, stmts[1], "foo")
}

func TestSplitStatements_TaggedDollarQuoting(t *testing.T) {
	sql := `CREATE FUNCTION foo() RETURNS void AS $fn$
BEGIN
  EXECUTE 'SELECT 1;';
END;
$fn$ LANGUAGE plpgsql;
SELECT 1`

	stmts := splitStatements(sql)

	require.Len(t, stmts, 2)
}

func TestSplitStatements_CreateTableWithTriggerColumnName(t *testing.T) {
	// "trigger" as a table name should NOT activate trigger mode
	sql := `CREATE TABLE trigger (id int); CREATE TABLE other (id int)`

	stmts := splitStatements(sql)

	require.Len(t, stmts, 2)
}

func TestSplitStatements_CreateIndexNotTrigger(t *testing.T) {
	sql := `CREATE INDEX idx ON my_table (col); SELECT 1`

	stmts := splitStatements(sql)

	require.Len(t, stmts, 2)
}

func TestSplitStatements_CreateTrigger(t *testing.T) {
	sql := `CREATE TRIGGER my_trigger AFTER INSERT ON t FOR EACH ROW BEGIN INSERT INTO log VALUES (NEW.id); END;
SELECT 1`

	stmts := splitStatements(sql)

	require.Len(t, stmts, 2)
	require.Contains(t, stmts[0], "my_trigger")
}

func TestRemoveInlineComments_PreservesDollarQuotedStrings(t *testing.T) {
	sql := `CREATE FUNCTION foo() AS $$
BEGIN
  x := 1; -- this is inside dollar quotes
END;
$$ LANGUAGE plpgsql`

	result := removeInlineComments(sql)

	require.Contains(t, result, "-- this is inside dollar quotes")
}

func TestRemoveInlineComments_StripsOutsideDollarQuotes(t *testing.T) {
	sql := "SELECT 1; -- outside comment\nSELECT 2"

	result := removeInlineComments(sql)

	require.NotContains(t, result, "outside comment")
	require.Contains(t, result, "SELECT 1")
	require.Contains(t, result, "SELECT 2")
}
