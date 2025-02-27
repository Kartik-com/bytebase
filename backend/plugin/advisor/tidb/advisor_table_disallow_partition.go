package tidb

// Framework code is generated by the generator.

import (
	"fmt"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableDisallowPartitionAdvisor)(nil)
	_ ast.Visitor     = (*tableDisallowPartitionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLTableDisallowPartition, &TableDisallowPartitionAdvisor{})
}

// TableDisallowPartitionAdvisor is the advisor checking for disallow table partition.
type TableDisallowPartitionAdvisor struct {
}

// Check checks for disallow table partition.
func (*TableDisallowPartitionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &tableDisallowPartitionChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.OriginTextPosition()
		(stmt).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type tableDisallowPartitionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (checker *tableDisallowPartitionChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisor.Ok
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		if node.Partition != nil {
			code = advisor.CreateTablePartition
		}
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			if spec.Tp == ast.AlterTablePartition {
				code = advisor.CreateTablePartition
				break
			}
		}
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", checker.text),
			Line:    checker.line,
		})
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*tableDisallowPartitionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
