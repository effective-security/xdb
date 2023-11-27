package schema

import (
	"testing"

	"github.com/effective-security/x/configloader"
	"github.com/effective-security/xdb/mocks/mockschema"
	"github.com/effective-security/xdb/pkg/cli/clisuite"
	dbschema "github.com/effective-security/xdb/schema"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

type testSuite struct {
	clisuite.TestSuite
}

func TestSchema(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) TestPrintColumnsCmd() {
	require := s.Require()

	ctrl := gomock.NewController(s.T())
	mock := mockschema.NewMockProvider(ctrl)
	s.Ctl.WithSchemaProvider(mock)

	res := dbschema.Tables{
		{
			Name:   "test",
			Schema: "dbo",
			Columns: dbschema.Columns{
				{
					Name:     "ID",
					Type:     "uint64",
					UdtType:  "int8",
					Nullable: false,
				},
			},
		},
	}

	mock.EXPECT().ListTables(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil).Times(2)
	mock.EXPECT().ListTables(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.Errorf("query failed")).Times(1)

	cmd := PrintColumnsCmd{
		DB:     "TestDb2",
		Schema: "dbo",
		Table:  []string{"Transaction"},
	}

	err := cmd.Run(s.Ctl)
	require.NoError(err)
	s.Equal("Schema: dbo\n"+
		"Table: test\n\n"+
		"  NAME |  TYPE  | UDT  | NULL | MAX | REF  \n"+
		"-------+--------+------+------+-----+------\n"+
		"  ID   | uint64 | int8 |      |     |      \n\n", s.Out.String())

	s.Ctl.O = "json"
	s.Out.Reset()

	err = cmd.Run(s.Ctl)
	require.NoError(err)
	s.Equal(
		"[\n  {\n    \"Schema\": \"dbo\",\n    \"Name\": \"test\",\n    \"IsView\": false,\n    \"Columns\": [\n      {\n        \"Name\": \"ID\",\n        \"Type\": \"uint64\",\n        \"UdtType\": \"int8\",\n        \"Nullable\": false,\n        \"MaxLength\": 0\n      }\n    ],\n    \"Indexes\": null,\n    \"PrimaryKey\": null\n  }\n]\n",
		s.Out.String())

	err = cmd.Run(s.Ctl)
	s.EqualError(err, "query failed")
}

func (s *testSuite) TestPrintTablesCmd() {
	require := s.Require()

	ctrl := gomock.NewController(s.T())
	mock := mockschema.NewMockProvider(ctrl)
	s.Ctl.WithSchemaProvider(mock)

	res := dbschema.Tables{
		{
			Name:   "test",
			Schema: "dbo",
			Columns: dbschema.Columns{
				{
					Name:     "ID",
					Type:     "numeric",
					Nullable: false,
				},
			},
		},
	}

	mock.EXPECT().ListTables(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil).Times(1)
	mock.EXPECT().ListTables(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.Errorf("query failed")).Times(1)

	cmd := PrintTablesCmd{
		DB:     "TestDb2",
		Schema: "dbo",
		Table:  []string{"Transaction"},
	}

	err := cmd.Run(s.Ctl)
	require.NoError(err)
	s.Equal("dbo.test\n", s.Out.String())

	err = cmd.Run(s.Ctl)
	s.EqualError(err, "query failed")
}

func (s *testSuite) TestPrintViewsCmd() {
	require := s.Require()

	ctrl := gomock.NewController(s.T())
	mock := mockschema.NewMockProvider(ctrl)
	s.Ctl.WithSchemaProvider(mock)

	res := dbschema.Tables{
		{
			Name:   "test",
			Schema: "dbo",
			Columns: dbschema.Columns{
				{
					Name:     "ID",
					Type:     "numeric",
					Nullable: false,
				},
			},
		},
	}

	mock.EXPECT().ListViews(gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil).Times(1)
	mock.EXPECT().ListViews(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.Errorf("query failed")).Times(1)

	cmd := PrintViewsCmd{
		DB:     "TestDb2",
		Schema: "dbo",
		View:   []string{"Transaction"},
	}

	err := cmd.Run(s.Ctl)
	require.NoError(err)
	s.Equal("dbo.test\n", s.Out.String())

	err = cmd.Run(s.Ctl)
	s.EqualError(err, "query failed")
}

func (s *testSuite) TestPrintFKCmd() {
	require := s.Require()

	ctrl := gomock.NewController(s.T())
	mock := mockschema.NewMockProvider(ctrl)
	s.Ctl.WithSchemaProvider(mock)

	res := dbschema.ForeignKeys{
		{
			Name:      "FK_1",
			Schema:    "dbo",
			Table:     "from",
			Column:    "col1",
			RefSchema: "dbo",
			RefTable:  "to",
			RefColumn: "col2",
		},
	}

	mock.EXPECT().ListForeignKeys(gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil).Times(2)
	mock.EXPECT().ListForeignKeys(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.Errorf("query failed")).Times(1)

	cmd := PrintFKCmd{
		DB:     "TestDb2",
		Schema: "dbo",
		Table:  []string{"Transaction"},
	}

	err := cmd.Run(s.Ctl)
	require.NoError(err)
	s.Equal(`  NAME | SCHEMA | TABLE | COLUMN | FK SCHEMA | FK TABLE | FK COLUMN  
-------+--------+-------+--------+-----------+----------+------------
  FK_1 | dbo    | from  | col1   | dbo       | to       | col2       

`, s.Out.String())

	s.Ctl.O = "json"
	s.Out.Reset()

	err = cmd.Run(s.Ctl)
	require.NoError(err)
	s.Equal("[\n  {\n    \"Name\": \"FK_1\",\n    \"Schema\": \"dbo\",\n    \"Table\": \"from\",\n    \"Column\": \"col1\",\n    \"RefSchema\": \"dbo\",\n    \"RefTable\": \"to\",\n    \"RefColumn\": \"col2\"\n  }\n]\n", s.Out.String())

	err = cmd.Run(s.Ctl)
	s.EqualError(err, "query failed")
}

func (s *testSuite) TestGenerate() {
	require := s.Require()

	var res dbschema.Tables
	err := configloader.Unmarshal("testdata/pg_columns.json", &res)
	require.NoError(err)

	cmd := GenerateCmd{
		PkgModel:  "model",
		PkgSchema: "schema",
		Schema:    "dbo",
		DB:        "testdb",
		Table:     []string{"Transaction"},
	}
	err = cmd.generate(s.Ctl, "postgres", "org", res)
	require.NoError(err)

	ctrl := gomock.NewController(s.T())
	mock := mockschema.NewMockProvider(ctrl)
	s.Ctl.WithSchemaProvider(mock)

	ret := dbschema.Tables{
		{
			Name:   "test",
			Schema: "dbo",
			Columns: dbschema.Columns{
				{
					Name:     "ID",
					Type:     "int8",
					Nullable: false,
				},
			},
		},
	}

	mock.EXPECT().Name().Return("postgres").Times(1)
	mock.EXPECT().ListTables(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(ret, nil).Times(1)
	err = cmd.Run(s.Ctl)
	require.NoError(err)
	s.HasText("DO NOT EDIT!", s.Out.String())
}
