package realtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── sendError / sendSuccess ──────────────────────────────────────────────────

func TestSendError(t *testing.T) {
	t.Run("writes an error-typed ServerMessage to the send channel", func(t *testing.T) {
		// sendError must put exactly one message on the buffered channel with
		// type="error" and the supplied payload string.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		c.sendError("something went wrong")

		data := <-c.send
		var msg ServerMessage
		assert.NoError(t, json.Unmarshal(data, &msg))
		assert.Equal(t, "error", msg.Type)
		assert.Equal(t, "something went wrong", msg.Payload)
	})
}

func TestSendSuccess(t *testing.T) {
	t.Run("writes a success-typed ServerMessage to the send channel", func(t *testing.T) {
		// sendSuccess must put exactly one message on the buffered channel with
		// type="success" and the supplied payload string.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		c.sendSuccess("operation completed")

		data := <-c.send
		var msg ServerMessage
		assert.NoError(t, json.Unmarshal(data, &msg))
		assert.Equal(t, "success", msg.Type)
		assert.Equal(t, "operation completed", msg.Payload)
	})
}

// ─── handleClientMessage ─────────────────────────────────────────────────────

func TestHandleClientMessage_InvalidJSON(t *testing.T) {
	t.Run("sends error when message is not valid JSON", func(t *testing.T) {
		// A non-JSON payload must be rejected with an error message before any
		// routing is attempted.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		c.handleClientMessage(context.Background(), []byte("not json"))

		data := <-c.send
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "error", msg.Type)
	})
}

func TestHandleClientMessage_UnknownStructure(t *testing.T) {
	t.Run("sends error when message has neither 'action' nor 'user_id' key", func(t *testing.T) {
		// A valid JSON object that doesn't match any known message shape must
		// produce an "Unknown message structure" error.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		c.handleClientMessage(context.Background(), []byte(`{"random_key":"value"}`))

		data := <-c.send
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "error", msg.Type)
		assert.Equal(t, "Unknown message structure", msg.Payload)
	})
}

func TestHandleClientMessage_RoutesToControlAction(t *testing.T) {
	t.Run("routes to handleControlAction when 'action' key is present", func(t *testing.T) {
		// A message containing "action" must be handled as a control command.
		// Successful subscribe sends a success reply on client.send.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		// Drain the subscribe channel so handleControlAction doesn't block.
		go func() { <-h.subscribe }()

		c.handleClientMessage(context.Background(), []byte(
			`{"action":"subscribe","items":[{"target_user_id":"u1","metric_type":"heart_rate"}]}`,
		))

		data := waitMsg(t, c.send)
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "success", msg.Type)
	})
}

func TestHandleClientMessage_RoutesToMetricPush(t *testing.T) {
	t.Run("routes to handleMetricPush when 'user_id' key is present", func(t *testing.T) {
		// A message containing "user_id" is treated as metric data from the device.
		// Attempting to push for a different user ID must produce a forbidden error.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "user-1")

		c.handleClientMessage(context.Background(), []byte(
			`{"user_id":"other-user","metric_type":"heart_rate","value":72}`,
		))

		data := waitMsg(t, c.send)
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "error", msg.Type)
		assert.Contains(t, msg.Payload.(string), "Forbidden")
	})
}

// ─── handleControlAction ─────────────────────────────────────────────────────

func TestHandleControlAction_InvalidJSON(t *testing.T) {
	t.Run("sends error when control message cannot be parsed", func(t *testing.T) {
		// A message that passes the raw-JSON check but is not a valid ClientMessage
		// struct must be rejected.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		c.handleControlAction(context.Background(), []byte(`{"action": bad json`))

		data := <-c.send
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "error", msg.Type)
	})
}

func TestHandleControlAction_ValidationError(t *testing.T) {
	t.Run("sends error when items array is empty (min=1 violated)", func(t *testing.T) {
		// The validator requires at least one SubscribeItem; an empty items array
		// must be rejected before any channel send.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		c.handleControlAction(context.Background(), []byte(`{"action":"subscribe","items":[]}`))

		data := <-c.send
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "error", msg.Type)
	})
}

func TestHandleControlAction_Subscribe_DispatchesEventAndSendsSuccess(t *testing.T) {
	t.Run("subscribe dispatches SubscriptionEvent to hub and replies with success", func(t *testing.T) {
		// For each item in the subscribe action a SubscriptionEvent must be sent
		// to hub.subscribe with the correct fields.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		var gotEvent SubscriptionEvent
		done := make(chan struct{})
		go func() {
			defer close(done)
			gotEvent = <-h.subscribe
		}()

		c.handleControlAction(context.Background(), []byte(
			`{"action":"subscribe","items":[{"target_user_id":"u1","metric_type":"heart_rate","group_id":"g1"}]}`,
		))
		<-done

		assert.Equal(t, c, gotEvent.Client)
		assert.Equal(t, "u1", gotEvent.TargetUserID)
		assert.Equal(t, "heart_rate", gotEvent.MetricType)
		assert.Equal(t, "g1", gotEvent.GroupID)

		data := <-c.send
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "success", msg.Type)
	})
}

func TestHandleControlAction_Unsubscribe_DispatchesEventAndSendsSuccess(t *testing.T) {
	t.Run("unsubscribe dispatches SubscriptionEvent to hub.unsubscribe and replies with success", func(t *testing.T) {
		// The unsubscribe action must send to hub.unsubscribe (not hub.subscribe)
		// and then reply with a success message.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		var gotEvent SubscriptionEvent
		done := make(chan struct{})
		go func() {
			defer close(done)
			gotEvent = <-h.unsubscribe
		}()

		c.handleControlAction(context.Background(), []byte(
			`{"action":"unsubscribe","items":[{"target_user_id":"u2","metric_type":"steps_count"}]}`,
		))
		<-done

		assert.Equal(t, "u2", gotEvent.TargetUserID)
		assert.Equal(t, "steps_count", gotEvent.MetricType)

		data := <-c.send
		var msg ServerMessage
		json.Unmarshal(data, &msg)
		assert.Equal(t, "success", msg.Type)
	})
}

func TestHandleControlAction_Subscribe_MultipleItems(t *testing.T) {
	t.Run("subscribing to multiple items dispatches one event per item", func(t *testing.T) {
		// Each item in the items array must generate its own SubscriptionEvent.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "viewer-1")

		events := make(chan SubscriptionEvent, 2)
		done := make(chan struct{})
		go func() {
			defer close(done)
			events <- <-h.subscribe
			events <- <-h.subscribe
		}()

		c.handleControlAction(context.Background(), []byte(`{
			"action":"subscribe",
			"items":[
				{"target_user_id":"u1","metric_type":"heart_rate"},
				{"target_user_id":"u1","metric_type":"steps_count"}
			]
		}`))
		<-done

		close(events)
		var metricTypes []string
		for ev := range events {
			metricTypes = append(metricTypes, ev.MetricType)
		}
		assert.ElementsMatch(t, []string{"heart_rate", "steps_count"}, metricTypes)
	})
}

// ─── handleMetricPush ─────────────────────────────────────────────────────────

func TestHandleMetricPush_ForbiddenUserID(t *testing.T) {
	t.Run("rejects push when UserID in payload does not match viewer's identity", func(t *testing.T) {
		// The security check must prevent spoofing: a client can only push data
		// for their own UserID.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "user-1")

		msg := `{"user_id":"other-user","metric_type":"heart_rate","value":72,"timestamp":"2026-01-01T00:00:00Z"}`
		c.handleMetricPush([]byte(msg))

		data := waitMsg(t, c.send)
		var srv ServerMessage
		json.Unmarshal(data, &srv)
		assert.Equal(t, "error", srv.Type)
		assert.Contains(t, srv.Payload.(string), "Forbidden")
	})
}

func TestHandleMetricPush_InvalidJSON(t *testing.T) {
	t.Run("sends error when metric payload is malformed JSON", func(t *testing.T) {
		// A non-parseable metric payload must be rejected with an error message.
		perm := new(mockPermRepo)
		h := newTestHub(perm)
		c := newTestClient(h, "user-1")

		c.handleMetricPush([]byte(`{invalid json`))

		data := waitMsg(t, c.send)
		var srv ServerMessage
		json.Unmarshal(data, &srv)
		assert.Equal(t, "error", srv.Type)
	})
}
