package tests

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-kallax.v1"
)

type QuerySuite struct {
	BaseTestSuite
}

func TestQuerySuite(t *testing.T) {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS query (
			id uuid primary key,
			idproperty uuid,
			idproperty_ptr uuid,
			inverse_id uuid,
			foo varchar(256),
			embedded jsonb,
			inline varchar(256),
			map_of_string jsonb,
			map_of_interface jsonb,
			map_of_some_type jsonb,
			string_property varchar(256),
			integer int,
			integer64 bigint,
			float32 float,
			boolean boolean,
			array_param text[],
			slice_param text[],
			alias_array_param text[],
			alias_slice_param text[],
			alias_string_param varchar(256),
			alias_int_param int,
			dummy_param jsonb,
			alias_dummy_param jsonb,
			slice_dummy_param jsonb,
			idproperty_param uuid,
			idproperty_ptr_param uuid,
			slice_idpptr_param uuid[],
			interface_prop_param text,
			urlparam varchar(256),
			time_param timestamptz,
			alias_arr_alias_string_param text[],
			alias_here_array_param text[],
			array_alias_here_string_param text[],
			scanner_valuer_param text
		)`,
		`CREATE TABLE IF NOT EXISTS query_relation (
			id uuid primary key,
			name  varchar(256),
			owner_id uuid references query(id)
		)`,
	}
	suite.Run(t, &QuerySuite{NewBaseSuite(schema, "query_relation", "query")})
}

func (s *QuerySuite) SetupTest() {
	s.BaseTestSuite.SetupTest()

	resetQueryFixtures()
	store := NewQueryFixtureStore(s.db)
	for _, fixture := range queryFixtures {
		s.Nil(store.Insert(fixture))
	}
}

func (s *QuerySuite) TestInsertTruncateTime() {
	s.BaseTestSuite.SetupTest()
	f := NewQueryFixture("fixture")
	for f.TimeParam.Nanosecond() == 0 {
		f.TimeParam = time.Now()
	}

	store := NewQueryFixtureStore(s.db)
	s.NoError(store.Insert(f))

	f2, err := store.FindOne(NewQueryFixtureQuery().FindByID(f.ID))
	s.NoError(err)
	s.Equal(f.TimeParam, f2.TimeParam.Local())
}

func (s *QuerySuite) TestUpdateTruncateTime() {
	s.BaseTestSuite.SetupTest()
	f := NewQueryFixture("fixture")
	store := NewQueryFixtureStore(s.db)
	s.NoError(store.Insert(f))
	for f.TimeParam.Nanosecond() == 0 {
		f.TimeParam = time.Now()
	}

	_, err := store.Update(f)
	s.NoError(err)
	f2, err := store.FindOne(NewQueryFixtureQuery().FindByID(f.ID))
	s.NoError(err)
	s.Equal(f.TimeParam, f2.TimeParam.Local())
}

func (s *QuerySuite) TestSaveTruncateTime() {
	s.BaseTestSuite.SetupTest()
	f := NewQueryFixture("fixture")
	for f.TimeParam.Nanosecond() == 0 {
		f.TimeParam = time.Now()
	}

	store := NewQueryFixtureStore(s.db)
	_, err := store.Save(f)
	s.NoError(err)

	f2, err := store.FindOne(NewQueryFixtureQuery().FindByID(f.ID))
	s.NoError(err)
	s.Equal(f.TimeParam, f2.TimeParam.Local())
}

func (s *QuerySuite) TestQuery() {
	store := NewQueryFixtureStore(s.db)
	doc := NewQueryFixture("bar")
	s.Nil(store.Insert(doc))

	query := NewQueryFixtureQuery()
	query.Where(kallax.Eq(Schema.QueryFixture.ID, doc.ID))

	s.NotPanics(func() {
		v, err := store.FindOne(query)
		s.NoError(err)
		s.Equal("bar",v.Foo)
	})

	notID := kallax.NewULID()
	queryErr := NewQueryFixtureQuery()
	queryErr.Where(kallax.Eq(Schema.QueryFixture.ID, notID))
	_, err := store.FindOne(queryErr)
	s.Error(err)
}

func (s *QuerySuite) TestFindById() {
	store := NewQueryFixtureStore(s.db)

	docName := "bar"
	doc := NewQueryFixture(docName)
	s.Nil(store.Insert(doc))

	query := NewQueryFixtureQuery()
	query.FindByID(doc.ID)
	s.NotPanics(func() {
		v, err := store.FindOne(query)
		s.NoError(err)
		s.Equal(docName, v.Foo)
	})

	queryManyId := NewQueryFixtureQuery()
	queryManyId.FindByID(queryFixtures[1].ID, queryFixtures[2].ID)
	count, err := store.Count(queryManyId)
	s.Equal(2, int(count))
	s.Nil(err)

	notID := kallax.NewULID()
	queryErr := NewQueryFixtureQuery()
	queryErr.FindByID(notID)

	_, err = store.FindOne(queryErr)
	s.Error(err)

}

func (s *QuerySuite) TestFindBy() {
	store := NewQueryFixtureStore(s.db)

	v, err := store.FindOne(NewQueryFixtureQuery().FindByStringProperty("StringProperty1"))
	s.NoError(err)
	s.True(v.Eq(queryFixtures[1]))


	_, err = store.FindOne(NewQueryFixtureQuery().FindByStringProperty("NOT_FOUND"))
	s.Error(err)

	v, err = store.FindOne(NewQueryFixtureQuery().FindByBoolean(false))
	s.NoError(err)
	s.True(v.Eq(queryFixtures[1]))

	count, err := store.Count(NewQueryFixtureQuery().FindByBoolean(true))
	s.Equal(int64(2), count)
	s.Nil(err)


	s.NotPanics(func() {
		v, err := store.FindOne(NewQueryFixtureQuery().FindByInteger(kallax.Eq, 2))
		s.NoError(err)
		s.True(v.Eq(queryFixtures[2]))
	})

	_, err = store.FindOne(NewQueryFixtureQuery().FindByInteger(kallax.Eq, 99))
	s.Error(err)

	s.NotPanics(func() {
		count, err := store.Count(NewQueryFixtureQuery().FindByInteger(kallax.GtOrEq, 1))
		s.Equal(int64(2), count)
		s.Nil(err)
	})
	s.NotPanics(func() {
		count, err := store.Count(NewQueryFixtureQuery().FindByInteger(kallax.Lt, 0))
		s.Equal(int64(0), count)
		s.Nil(err)
	})
}

func (s *QuerySuite) TestGeneration() {
	var cases = []struct {
		propertyName        string
		autoGeneratedFindBy bool
	}{
		{"ID", true},
		{"SelfRelation", false},
		{"Inverse", true},
		{"SelfNRelation", false},
		{"Embedded", false},
		{"Ignored", false},
		{"Inline", true},
		{"MapOfString", false},
		{"MapOfInterface", false},
		{"MapOfSomeType", false},
		{"Foo", true},
		{"StringProperty", true},
		{"Integer", true},
		{"Integer64", true},
		{"Float32", true},
		{"Boolean", true},
		{"ArrayParam", true},
		{"SliceParam", true},
		{"AliasArrayParam", true},
		{"AliasSliceParam", true},
		{"AliasStringParam", true},
		{"AliasIntParam", true},
		{"DummyParam", false},
		{"AliasDummyParam", false},
		{"SliceDummyParam", false},
		{"IDPropertyParam", true},
		{"InterfacePropParam", true},
		{"URLParam", true},
		{"TimeParam", true},
		{"AliasArrAliasStringParam", true},
		{"AliasArrAliasSliceParam", false},
		{"ArrayArrayParam", false},
		{"AliasHereArrayParam", true},
		{"ScannerValuerParam", true},
	}

	q := NewQueryFixtureQuery()
	for _, c := range cases {
		s.hasFindByMethod(q, c.propertyName, c.autoGeneratedFindBy)
	}
}

func (s *QuerySuite) hasFindByMethod(q *QueryFixtureQuery, name string, exists bool) {
	queryValue := reflect.TypeOf(q)
	_, ok := queryValue.MethodByName("FindBy" + name)
	if exists {
		s.True(ok, "'FindBy%s' method should BE generated", name)
	} else {
		s.False(ok, "'FindBy%s' method should NOT be generated", name)
	}
}
