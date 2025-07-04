package capytest

import "testing"

type testSuite struct {
	t          *testing.T
	p          Provider
	beforeEach func(t *testing.T, r Runner)
}

type TestSuite interface {
	Run(name string, f func(t *testing.T, r Runner))
	BeforeEach(f func(t *testing.T, r Runner))
}

func NewTestSuite(t *testing.T, p Provider) TestSuite {
	return &testSuite{t, p, nil}
}

func (s *testSuite) runner(t *testing.T) Runner {
	s.t.Helper()

	if prep, ok := s.p.(PreparableProvider); ok {
		if err := prep.Prepare(); err != nil {
			t.Fatalf("failed to prepare provider: %v", err)
		}
		t.Cleanup(func() {
			if err := prep.Cleanup(); err != nil {
				t.Errorf("failed to cleanup provider: %v", err)
			}
		})
	}

	return NewRunner(s.p)
}

func (s *testSuite) BeforeEach(f func(t *testing.T, r Runner)) {
	s.beforeEach = f
}

func (s *testSuite) Run(name string, f func(t *testing.T, r Runner)) {
	s.t.Helper()
	s.t.Run(name, func(t *testing.T) {
		r := s.runner(t)
		if s.beforeEach != nil {
			s.beforeEach(t, r)
		}
		f(t, r)
	})
}
