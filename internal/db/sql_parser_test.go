package db

import (
	"strings"
	"testing"
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

	if len(stmts) != 2 {
		failNowf(t, "expected 2 statements, got %d: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "notify_change") {
		failf(t, "first statement should contain the full function body, got: %s", stmts[0])
	}
	if !strings.Contains(stmts[1], "foo") {
		failf(t, "second statement should be CREATE TABLE, got: %s", stmts[1])
	}
}

func TestSplitStatements_TaggedDollarQuoting(t *testing.T) {
	sql := `CREATE FUNCTION foo() RETURNS void AS $fn$
BEGIN
  EXECUTE 'SELECT 1;';
END;
$fn$ LANGUAGE plpgsql;
SELECT 1`

	stmts := splitStatements(sql)

	if len(stmts) != 2 {
		failNowf(t, "expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitStatements_CreateTableWithTriggerColumnName(t *testing.T) {
	// "trigger" as a table name should NOT activate trigger mode
	sql := `CREATE TABLE trigger (id int); CREATE TABLE other (id int)`

	stmts := splitStatements(sql)

	if len(stmts) != 2 {
		failNowf(t, "expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitStatements_CreateIndexNotTrigger(t *testing.T) {
	sql := `CREATE INDEX idx ON my_table (col); SELECT 1`

	stmts := splitStatements(sql)

	if len(stmts) != 2 {
		failNowf(t, "expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitStatements_CreateTrigger(t *testing.T) {
	sql := `CREATE TRIGGER my_trigger AFTER INSERT ON t FOR EACH ROW BEGIN INSERT INTO log VALUES (NEW.id); END;
SELECT 1`

	stmts := splitStatements(sql)

	if len(stmts) != 2 {
		failNowf(t, "expected 2 statements, got %d: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "my_trigger") {
		failf(t, "first statement should contain trigger, got: %s", stmts[0])
	}
}

func TestRemoveInlineComments_PreservesDollarQuotedStrings(t *testing.T) {
	sql := `CREATE FUNCTION foo() AS $$
BEGIN
  x := 1; -- this is inside dollar quotes
END;
$$ LANGUAGE plpgsql`

	result := removeInlineComments(sql)

	if !strings.Contains(result, "-- this is inside dollar quotes") {
		failf(t, "comment inside dollar-quoted string should be preserved, got: %s", result)
	}
}

func TestRemoveInlineComments_StripsOutsideDollarQuotes(t *testing.T) {
	sql := "SELECT 1; -- outside comment\nSELECT 2"

	result := removeInlineComments(sql)

	if strings.Contains(result, "outside comment") {
		failf(t, "comment outside dollar quotes should be removed, got: %s", result)
	}
	if !strings.Contains(result, "SELECT 1") || !strings.Contains(result, "SELECT 2") {
		failf(t, "SQL statements should be preserved, got: %s", result)
	}
}
