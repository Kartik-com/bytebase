package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestGetSQLStatementPrefix(t *testing.T) {
	tests := []struct {
		engine       storepb.Engine
		resourceList []base.SchemaResource
		columnNames  []string
		want         string
	}{
		{
			engine:       storepb.Engine_MYSQL,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO `<table_name>` (`a`) VALUES (",
		},
		{
			engine:       storepb.Engine_MYSQL,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a", "b"},
			want:         "INSERT INTO `table1` (`a`,`b`) VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: nil,
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"<table_name>\" (\"a\") VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "", Table: "table1"}},
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"table1\" (\"a\") VALUES (",
		},
		{
			engine:       storepb.Engine_POSTGRES,
			resourceList: []base.SchemaResource{{Database: "db1", Schema: "schema1", Table: "table1"}},
			columnNames:  []string{"a"},
			want:         "INSERT INTO \"schema1\".\"table1\" (\"a\") VALUES (",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		got, err := getSQLStatementPrefix(test.engine, test.resourceList, test.columnNames)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

func TestExportSQL(t *testing.T) {
	tests := []struct {
		engine          storepb.Engine
		statementPrefix string
		result          *v1pb.QueryResult
		want            string
	}{
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BoolValue{BoolValue: true},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "abc"},
							},
							{
								Kind: &v1pb.RowValue_NullValue{},
							},
						},
					},
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_BoolValue{BoolValue: false},
							},
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "abc"},
							},
							{
								Kind: &v1pb.RowValue_NullValue{},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES (true,'abc',NULL);\nINSERT INTO `<table_name>` (`a`) VALUES (false,'abc',NULL);",
		},
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\nbc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\\nbc');",
		},
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a'b"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a''b');",
		},
		{
			engine:          storepb.Engine_MYSQL,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\b"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\\b');",
		},
		{
			engine:          storepb.Engine_POSTGRES,
			statementPrefix: "INSERT INTO `<table_name>` (`a`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\nbc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`a`) VALUES ('a\nbc');",
		},
		{
			engine:          storepb.Engine_POSTGRES,
			statementPrefix: "INSERT INTO `<table_name>` (`b`) VALUES (",
			result: &v1pb.QueryResult{
				Rows: []*v1pb.QueryRow{
					{
						Values: []*v1pb.RowValue{
							{
								Kind: &v1pb.RowValue_StringValue{StringValue: "a\\bc"},
							},
						},
					},
				},
			},
			want: "INSERT INTO `<table_name>` (`b`) VALUES ( E'a\\\\bc');",
		},
	}
	a := assert.New(t)

	for _, test := range tests {
		got, err := exportSQL(test.engine, test.statementPrefix, test.result)
		a.NoError(err)
		a.Equal(test.want, string(got))
	}
}

func TestEncodeToBase64String(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "",
			want:  "",
		},
		{
			input: "select * from employee",
			want:  "c2VsZWN0ICogZnJvbSBlbXBsb3llZQ==",
		},
		{
			input: "select name as 姓名 from employee",
			want:  "c2VsZWN0IG5hbWUgYXMg5aeT5ZCNIGZyb20gZW1wbG95ZWU=",
		},
		{
			input: "Hello 哈喽 👋",
			want:  "SGVsbG8g5ZOI5Za9IPCfkYs=",
		},
	}

	for _, test := range tests {
		got := encodeToBase64String(test.input)
		if got != test.want {
			t.Errorf("encodeToBase64String(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestGetExcelColumnName(t *testing.T) {
	a := assert.New(t)

	tests := []struct {
		index int
		want  string
	}{
		{
			index: 0,
			want:  "A",
		},
		{
			index: 3,
			want:  "D",
		},
		{
			index: 25,
			want:  "Z",
		},
		{
			index: 26,
			want:  "AA",
		},
		{
			index: 27,
			want:  "AB",
		},
		{
			index: excelMaxColumn - 1,
			want:  "ZZZ",
		},
	}

	for _, test := range tests {
		got, err := getExcelColumnName(test.index)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}

func TestEvalMaskingLevelOfColumn(t *testing.T) {
	defaultDatabaseMessage := &store.DatabaseMessage{
		EffectiveEnvironmentID: "prod",
		ProjectID:              "bytebase",
		InstanceID:             "neon-host",
		DatabaseName:           "bb",
	}

	defaultProjectDatabaseDataClassificationID := "2b599739-41da-4c35-a9ff-4a73c6cfe32c"

	defaultClassification := &storepb.DataClassificationSetting{
		Configs: []*storepb.DataClassificationSetting_DataClassificationConfig{
			{
				Id: defaultProjectDatabaseDataClassificationID,
				Levels: []*storepb.DataClassificationSetting_DataClassificationConfig_Level{
					{
						Id: "S1",
					},
					{
						Id: "S2",
					},
				},
				Classification: map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification{
					"1-1-1": {
						Id:    "1-1-1",
						Title: "personal",
						LevelId: func() *string {
							a := "S2"
							return &a
						}(),
					},
				},
			},
		},
	}

	testCases := []struct {
		description                             string
		databaseMessage                         *store.DatabaseMessage
		databaseProjectDatabaseClassificationID string
		schemaName                              string
		tableName                               string
		columnName                              string
		columnClassification                    string
		maskingPolicyMap                        map[maskingPolicyKey]*storepb.MaskData
		maskingRulePolicy                       *storepb.MaskingRulePolicy
		filteredMaskingExceptions               []*storepb.MaskingExceptionPolicy_MaskingException
		dataClassification                      *storepb.DataClassificationSetting

		want storepb.MaskingLevel
	}{
		{
			description:          "Follow The Global Masking Rule If Column Masking Policy Is Default",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap:     map[maskingPolicyKey]*storepb.MaskData{},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
				},
			},
			filteredMaskingExceptions:               []*storepb.MaskingExceptionPolicy_MaskingException{},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_FULL,
		},
		{
			description:          "Follow The Global Masking Rule If Column Masking Policy Is Default And Respect The Exception",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap:     map[maskingPolicyKey]*storepb.MaskData{},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						MaskingLevel: storepb.MaskingLevel_FULL,
					},
				},
			},
			filteredMaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
				{
					Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Member:       "zp@bytebase.com",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_PARTIAL,
		},
		{
			description:          "Only Find The Lower Level in Exception",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap:     map[maskingPolicyKey]*storepb.MaskData{},
			maskingRulePolicy: &storepb.MaskingRulePolicy{
				Rules: []*storepb.MaskingRulePolicy_MaskingRule{
					{
						// Classification hit.
						Condition:    &expr.Expr{Expression: `(table_name == "no_table") || (classification_level == "S2")`},
						MaskingLevel: storepb.MaskingLevel_PARTIAL,
					},
				},
			},
			filteredMaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
				{
					// Hit, but MaskingLevel_FULL > MaskingLevel_PARTIAL, do not replace the rule.
					Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Member:       "zp@bytebase.com",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_PARTIAL,
		},
		{
			description:          "Respect The Column Masking Policy",
			databaseMessage:      defaultDatabaseMessage,
			schemaName:           "hiring",
			tableName:            "employees",
			columnName:           "salary",
			columnClassification: "1-1-1",
			maskingPolicyMap: map[maskingPolicyKey]*storepb.MaskData{
				{
					schema: "hiring",
					table:  "employees",
					column: "salary",
				}: {
					Schema:       "hiring",
					Table:        "employees",
					Column:       "salary",
					MaskingLevel: storepb.MaskingLevel_FULL,
				},
			},
			maskingRulePolicy: &storepb.MaskingRulePolicy{},
			filteredMaskingExceptions: []*storepb.MaskingExceptionPolicy_MaskingException{
				{
					// Hit, and MaskingLevel_PARTIAL < MaskingLevel_FULL.
					Action: storepb.MaskingExceptionPolicy_MaskingException_QUERY,
					Condition: &expr.Expr{
						Expression: `(resource.instance_id == "neon-host") && (resource.database_name == "bb") && (resource.schema_name == "hiring") && (resource.table_name == "employees") && (resource.column_name == "salary")`,
					},
					Member:       "zp@bytebase.com",
					MaskingLevel: storepb.MaskingLevel_PARTIAL,
				},
			},
			dataClassification:                      defaultClassification,
			databaseProjectDatabaseClassificationID: defaultProjectDatabaseDataClassificationID,

			want: storepb.MaskingLevel_PARTIAL,
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		m := newEmptyMaskingLevelEvaluator().withMaskingRulePolicy(tc.maskingRulePolicy).withDataClassificationSetting(tc.dataClassification)
		result, err := m.evaluateMaskingLevelOfColumn(tc.databaseMessage, tc.schemaName, tc.tableName, tc.columnName, tc.columnClassification, tc.databaseProjectDatabaseClassificationID, tc.maskingPolicyMap, tc.filteredMaskingExceptions)
		a.NoError(err, tc.description)
		a.Equal(tc.want, result, tc.description)
	}
}
