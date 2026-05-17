package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"realtime-service/internal/metric"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ─── shared test helpers ──────────────────────────────────────────────────────

// mockPermRepo is an inline testify mock for permission.Repository.
type mockPermRepo struct{ mock.Mock }

func (m *mockPermRepo) CheckAccess(ctx context.Context, observerUserID, targetUserID, metricType string) (bool, error) {
	ret := m.Called(ctx, observerUserID, targetUserID, metricType)
	return ret.Bool(0), ret.Error(1)
}

func (m *mockPermRepo) GetMetricWatchers(ctx context.Context, targetUserID, metricType string) (map[string][]string, error) {
	ret := m.Called(ctx, targetUserID, metricType)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(map[string][]string), ret.Error(1)
}

// newTestHub creates a Hub with the given permRepo (no Kafka producer needed).
func newTestHub(perm *mockPermRepo) *Hub {
	return NewHub(perm, nil)
}

// newTestClient creates a Client with a buffered send channel and no real WebSocket.
func newTestClient(h *Hub, viewerID string) *Client {
	return &Client{
		id:       "test-" + viewerID,
		hub:      h,
		conn:     nil,
		send:     make(chan []byte, 256),
		viewerId: viewerID,
	}
}

// waitMsg waits up to 150 ms for a message on ch, failing the test on timeout.
func waitMsg(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(150 * time.Millisecond):
		t.Fatal("timeout waiting for message on send channel")
		return nil
	}
}

// waitClosed waits up to 150 ms for ch to be closed, failing on timeout.
func waitClosed(t *testing.T, ch <-chan []byte) {
	t.Helper()
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "expected channel to be closed, got a value instead")
	case <-time.After(150 * time.Millisecond):
		t.Fatal("timeout waiting for send channel to close")
	}
}

// ─── removeClientSubscriptions ────────────────────────────────────────────────

func TestRemoveClientSubscriptions(t *testing.T) {
	t.Run("removes target client while leaving other clients intact", func(t *testing.T) {
		// Two clients share the same group bucket; removing one must leave the other.
		h := &Hub{subscriptions: make(map[string]map[string]map[string]map[*Client]bool)}
		c1 := &Client{id: "c1"}
		c2 := &Client{id: "c2"}
		h.subscriptions["u1"] = map[string]map[string]map[*Client]bool{
			"heart_rate": {"g1": {c1: true, c2: true}},
		}

		h.removeClientSubscriptions(c1)

		assert.False(t, h.subscriptions["u1"]["heart_rate"]["g1"][c1])
		assert.True(t, h.subscriptions["u1"]["heart_rate"]["g1"][c2])
	})

	t.Run("prunes all empty ancestor maps after the last client is removed", func(t *testing.T) {
		// When the last client in a group bucket is removed, all empty parent
		// maps (group → metric → user) must be pruned too.
		h := &Hub{subscriptions: make(map[string]map[string]map[string]map[*Client]bool)}
		c := &Client{id: "only"}
		h.subscriptions["u1"] = map[string]map[string]map[*Client]bool{
			"heart_rate": {"g1": {c: true}},
		}

		h.removeClientSubscriptions(c)

		assert.Empty(t, h.subscriptions)
	})

	t.Run("no-op when the client has no subscriptions", func(t *testing.T) {
		// Removing an unknown client from an empty subscription tree must not panic.
		h := &Hub{subscriptions: make(map[string]map[string]map[string]map[*Client]bool)}

		assert.NotPanics(t, func() { h.removeClientSubscriptions(&Client{id: "ghost"}) })
	})

	t.Run("client subscribed to multiple metrics cleaned completely", func(t *testing.T) {
		// A client subscribed to two different metrics must be removed from both.
		h := &Hub{subscriptions: make(map[string]map[string]map[string]map[*Client]bool)}
		c := &Client{id: "multi"}
		h.subscriptions["u1"] = map[string]map[string]map[*Client]bool{
			"heart_rate":  {"g1": {c: true}},
			"steps_count": {"g1": {c: true}},
		}

		h.removeClientSubscriptions(c)

		assert.Empty(t, h.subscriptions)
	})
}

// ─── Hub.Run — register / unregister ──────────────────────────────────────────

func TestHub_Unregister_ClosesClientSend(t *testing.T) {
	// After unregistering, the hub must close the client's send channel so that
	// any writePump goroutine waiting on it knows to exit.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "viewer-1")
	h.register <- c
	h.unregister <- c

	waitClosed(t, c.send)
}

func TestHub_Unregister_WithSubscriptions_CleansMap(t *testing.T) {
	// Unregistering a client that has active subscriptions must remove all of them
	// so the hub doesn't accumulate stale entries.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "viewer-sub")

	// Subscribe then unregister
	h.register <- c
	h.subscribe <- SubscriptionEvent{Client: c, TargetUserID: "t1", MetricType: "heart_rate", GroupID: "g1"}
	h.unregister <- c

	waitClosed(t, c.send)
	// After send is closed the hub has finished processing; subscriptions should be empty.
	// Access subscriptions after hub stops to avoid race.
	cancel()
	time.Sleep(10 * time.Millisecond)
	assert.Empty(t, h.subscriptions)
}

// ─── Hub.Run — subscribe / unsubscribe ────────────────────────────────────────

func TestHub_Subscribe_ThenBroadcast_DeliverToAuthorizedClient(t *testing.T) {
	// A subscribed client whose viewerId matches the authorization list must
	// receive the broadcast message wrapped in a "metric" ServerMessage.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "watcher-1")
	h.subscribe <- SubscriptionEvent{
		Client:       c,
		TargetUserID: "target-1",
		MetricType:   "heart_rate",
		GroupID:      "group-1",
	}

	perm.On("GetMetricWatchers", mock.Anything, "target-1", "heart_rate").
		Return(map[string][]string{"watcher-1": {"group-1"}}, nil).Once()

	h.broadcast <- &metric.HealthMetric{UserID: "target-1", Type: "heart_rate", Value: 72}

	data := waitMsg(t, c.send)
	var msg ServerMessage
	assert.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "metric", msg.Type)
	perm.AssertExpectations(t)
}

func TestHub_Unsubscribe_PreventsDelivery(t *testing.T) {
	// After unsubscribing, no broadcast for the same (target, metric, group) tuple
	// must be delivered — even when the permission repo authorizes the viewer.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "watcher-1")
	h.subscribe <- SubscriptionEvent{Client: c, TargetUserID: "t1", MetricType: "heart_rate", GroupID: "g1"}
	h.unsubscribe <- SubscriptionEvent{Client: c, TargetUserID: "t1", MetricType: "heart_rate", GroupID: "g1"}

	perm.On("GetMetricWatchers", mock.Anything, "t1", "heart_rate").
		Return(map[string][]string{"watcher-1": {"g1"}}, nil).Maybe()

	h.broadcast <- &metric.HealthMetric{UserID: "t1", Type: "heart_rate", Value: 72}

	select {
	case <-c.send:
		t.Fatal("client received a message after unsubscribing")
	case <-time.After(60 * time.Millisecond):
		// expected: no delivery
	}
}

// ─── Hub.Run — broadcast authorization logic ──────────────────────────────────

func TestHub_Broadcast_NoWatchers_NoDelivery(t *testing.T) {
	// When GetMetricWatchers returns an empty map, nothing must be sent to any client.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "watcher-1")
	h.subscriptions["target-1"] = map[string]map[string]map[*Client]bool{
		"heart_rate": {"g1": {c: true}},
	}

	perm.On("GetMetricWatchers", mock.Anything, "target-1", "heart_rate").
		Return(map[string][]string{}, nil).Once()

	h.broadcast <- &metric.HealthMetric{UserID: "target-1", Type: "heart_rate"}

	select {
	case <-c.send:
		t.Fatal("expected no delivery when no watchers")
	case <-time.After(60 * time.Millisecond):
		// expected
	}
	perm.AssertExpectations(t)
}

func TestHub_Broadcast_UnauthorizedViewer_NoDelivery(t *testing.T) {
	// A subscribed client whose viewerId is not in the authorized watcher map
	// must not receive the broadcast.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "unauthorized-viewer")
	h.subscriptions["target-1"] = map[string]map[string]map[*Client]bool{
		"heart_rate": {"g1": {c: true}},
	}

	perm.On("GetMetricWatchers", mock.Anything, "target-1", "heart_rate").
		Return(map[string][]string{"other-viewer": {"g1"}}, nil).Once()

	h.broadcast <- &metric.HealthMetric{UserID: "target-1", Type: "heart_rate"}

	select {
	case <-c.send:
		t.Fatal("unauthorized viewer received a message")
	case <-time.After(60 * time.Millisecond):
		// expected
	}
	perm.AssertExpectations(t)
}

func TestHub_Broadcast_GlobalGroupSubscription_AuthorizedByAnyGroup(t *testing.T) {
	// A client subscribed with groupID="" (global) must receive a broadcast if
	// the permission repo authorizes them via ANY group.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "global-watcher")
	h.subscribe <- SubscriptionEvent{
		Client:       c,
		TargetUserID: "target-1",
		MetricType:   "heart_rate",
		GroupID:      "", // global
	}

	perm.On("GetMetricWatchers", mock.Anything, "target-1", "heart_rate").
		Return(map[string][]string{"global-watcher": {"any-group"}}, nil).Once()

	h.broadcast <- &metric.HealthMetric{UserID: "target-1", Type: "heart_rate", Value: 80}

	data := waitMsg(t, c.send)
	var msg ServerMessage
	assert.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "metric", msg.Type)
	perm.AssertExpectations(t)
}

func TestHub_Broadcast_GroupSubscription_RequiresMatchingGroup(t *testing.T) {
	// A client subscribed to groupID="g1" must NOT receive a message when they
	// are only authorized via a different group "g2".
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "watcher-1")
	h.subscribe <- SubscriptionEvent{
		Client:       c,
		TargetUserID: "target-1",
		MetricType:   "heart_rate",
		GroupID:      "g1",
	}

	// authorized only via g2, not g1
	perm.On("GetMetricWatchers", mock.Anything, "target-1", "heart_rate").
		Return(map[string][]string{"watcher-1": {"g2"}}, nil).Once()

	h.broadcast <- &metric.HealthMetric{UserID: "target-1", Type: "heart_rate", Value: 80}

	select {
	case <-c.send:
		t.Fatal("client received message despite being authorized by wrong group")
	case <-time.After(60 * time.Millisecond):
		// expected
	}
	perm.AssertExpectations(t)
}

func TestHub_Broadcast_PermRepoError_DroppedSilently(t *testing.T) {
	// When GetMetricWatchers returns an error, the hub must log it and continue
	// (no panic, no delivery).
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(h, "watcher-1")
	h.subscriptions["target-1"] = map[string]map[string]map[*Client]bool{
		"heart_rate": {"g1": {c: true}},
	}

	perm.On("GetMetricWatchers", mock.Anything, "target-1", "heart_rate").
		Return(nil, errors.New("db connection refused")).Once()

	h.broadcast <- &metric.HealthMetric{UserID: "target-1", Type: "heart_rate"}

	select {
	case <-c.send:
		t.Fatal("message delivered despite permission repository error")
	case <-time.After(60 * time.Millisecond):
		// expected
	}
	perm.AssertExpectations(t)
}

func TestHub_Broadcast_NoSubscribersForMetric_NoDelivery(t *testing.T) {
	// A broadcast for heart_rate must not reach a client subscribed only to
	// steps_count, even though GetMetricWatchers returns an authorized watcher.
	// The hub calls GetMetricWatchers first, then gates delivery on the local
	// subscriptions map — so no matching subscription means no delivery.
	perm := new(mockPermRepo)
	h := newTestHub(perm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	// subscriptions exist for a different metric type
	c := newTestClient(h, "watcher-1")
	h.subscriptions["target-1"] = map[string]map[string]map[*Client]bool{
		"steps_count": {"g1": {c: true}}, // different metric
	}

	// hub calls GetMetricWatchers before checking subscriptions; provide a
	// non-empty result so execution reaches the subscription-gating logic.
	perm.On("GetMetricWatchers", mock.Anything, "target-1", "heart_rate").
		Return(map[string][]string{"watcher-1": {"g1"}}, nil).Maybe()

	// broadcast is for heart_rate — no subscriber
	h.broadcast <- &metric.HealthMetric{UserID: "target-1", Type: "heart_rate"}

	select {
	case <-c.send:
		t.Fatal("unexpected delivery with no matching subscriptions")
	case <-time.After(60 * time.Millisecond):
		// expected
	}
}
