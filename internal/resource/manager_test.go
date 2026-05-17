package resource

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"carbon/domain"
	"carbon/remote"
)

// fakeClient implements remote.Client. Only GetResources is used by Manager;
// the rest panic so accidental use is obvious in tests.
type fakeClient struct {
	resources []domain.Resource
	err       error
	calls     int32
}

func (f *fakeClient) GetResources(ctx context.Context) ([]domain.Resource, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.err != nil {
		return nil, f.err
	}
	return f.resources, nil
}

func (*fakeClient) GetResource(context.Context, string) (domain.Resource, error) {
	panic("unused")
}
func (*fakeClient) GetResourceCategories(context.Context) ([]domain.ResourceCategory, remote.TreeMap, error) {
	panic("unused")
}
func (*fakeClient) GetResourceCategory(context.Context) (domain.ResourceCategory, error) {
	panic("unused")
}
func (*fakeClient) GetResourceReviews(context.Context, string) ([]domain.ResourceReview, error) {
	panic("unused")
}
func (*fakeClient) GetResourceVersions(context.Context, string) ([]domain.ResourceVersion, error) {
	panic("unused")
}
func (*fakeClient) GetResourceVersion(context.Context, string) (domain.ResourceVersion, error) {
	panic("unused")
}
func (*fakeClient) GetUser(context.Context, int) (domain.User, error) { panic("unused") }
func (*fakeClient) ValidateUserAuthCredentials(context.Context, interface{}) (remote.RawUserAuthResponse, error) {
	panic("unused")
}

func TestNewManager_PopulatesCacheFromRemote(t *testing.T) {
	fc := &fakeClient{resources: []domain.Resource{
		{ResourceId: 1, Title: "a"},
		{ResourceId: 2, Title: "b"},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := NewManager(ctx, fc)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	got := m.Collection()
	if len(got) != 2 {
		t.Fatalf("Collection len = %d, want 2", len(got))
	}
	if got[0].ResourceId != 1 || got[1].ResourceId != 2 {
		t.Errorf("unexpected collection: %+v", got)
	}
}

func TestNewManager_RemoteErrorBubblesUp(t *testing.T) {
	wantErr := errors.New("boom")
	fc := &fakeClient{err: wantErr}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := NewManager(ctx, fc); !errors.Is(err, wantErr) {
		t.Errorf("NewManager err = %v, want %v", err, wantErr)
	}
}

func TestManager_PutReplacesCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := NewManager(ctx, &fakeClient{resources: []domain.Resource{{ResourceId: 1}}})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	m.Put([]*domain.Resource{{ResourceId: 42}, {ResourceId: 43}})

	got := m.Collection()
	if len(got) != 2 || got[0].ResourceId != 42 {
		t.Errorf("Put did not replace cache: %+v", got)
	}
}

func TestManager_Add(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := NewManager(ctx, &fakeClient{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	m.Add(&domain.Resource{ResourceId: 10})
	m.Add(&domain.Resource{ResourceId: 11})

	if len(m.Collection()) != 2 {
		t.Errorf("Add did not append: %d", len(m.Collection()))
	}
}

func TestManager_FindReturnsMatchOrNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := NewManager(ctx, &fakeClient{resources: []domain.Resource{
		{ResourceId: 5, Title: "five"},
		{ResourceId: 6, Title: "six"},
	}})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	got := m.Find(func(r *domain.Resource) bool { return r.ResourceId == 6 })
	if got == nil || got.Title != "six" {
		t.Errorf("Find did not return expected resource: %+v", got)
	}

	miss := m.Find(func(r *domain.Resource) bool { return r.ResourceId == 999 })
	if miss != nil {
		t.Errorf("Find should return nil for no match, got %+v", miss)
	}
}

func TestManager_StartStopsOnContextCancel(t *testing.T) {
	// Smoke test that the background goroutine exits when ctx is cancelled.
	// We can't easily observe goroutine exit, but cancellation should not
	// leak or panic. The Start ticker is 5 minutes so we won't see a refresh.
	ctx, cancel := context.WithCancel(context.Background())
	m, err := NewManager(ctx, &fakeClient{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	cancel()
	// Give the goroutine a chance to notice cancellation.
	time.Sleep(10 * time.Millisecond)
	// Cache should still be readable after cancel.
	_ = m.Collection()
}
