package api

import (
	"context"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/follow"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

var (
	activityIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
	// Live Activity push tokens are longer than device tokens (128 bytes in the simulator).
	liveTokenRe = regexp.MustCompile(`^[0-9a-fA-F]{64,512}$`)
)

// requestFollow is a request followed from this app. The token is never returned.
type requestFollow struct {
	RequestID  string `json:"requestId"`
	Platform   string `json:"platform" enum:"ios,android"`
	Stage      string `json:"stage" enum:"requested,approved,searching,downloading,available,declined,failed"`
	Percent    int    `json:"percent" minimum:"0" maximum:"100"`
	ETASeconds int    `json:"etaSeconds" minimum:"0" doc:"0 = unknown"`
	UpdatedAt  int64  `json:"updatedAt" doc:"Unix seconds the state was computed"`
	ExpiresAt  int64  `json:"expiresAt" doc:"Unix seconds the follow ends on its own"`
}

type followList struct {
	Items []requestFollow `json:"items"`
}

type followRequestInput struct {
	ID   string `path:"id" maxLength:"32"`
	Body struct {
		PushToken  string `json:"pushToken" minLength:"20" maxLength:"1024" doc:"iOS: the Live Activity push token (hex). Android: the current FCM registration token"`
		ActivityID string `json:"activityId,omitempty" maxLength:"64" doc:"iOS only: the ActivityKit activity id. A new one restarts the follow"`
		Sandbox    bool   `json:"sandbox,omitempty" doc:"iOS only: the token comes from a development build (APNs sandbox)"`
		Server     string `json:"server,omitempty" maxLength:"255" doc:"Host (and port) the app uses to reach this server"`
	}
}

type followIDInput struct {
	ID string `path:"id" maxLength:"32"`
}

func registerFollows(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "followRequest", Method: http.MethodPut, Path: "/requests/{id}/follow",
		Summary: "Follow a request from this app", Tags: []string{"requests"}, Security: sec,
		Description: "Pushes the request's stage and download progress to a Live Activity (iOS) or an ongoing notification (Android) until it is available, declined or failed. Call it again whenever the token changes. Answers the current state.",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *followRequestInput) (*struct{ Body requestFollow }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		sess := sessionFrom(ctx)
		if sess == nil {
			return nil, fail(401, "login_required", "login required")
		}
		tok := strings.TrimSpace(in.Body.PushToken)
		switch sess.Platform {
		case "ios":
			if !liveTokenRe.MatchString(tok) {
				return nil, fail(422, "invalid_push_token", "not a Live Activity push token")
			}
		case "android":
			if !fcmTokenRe.MatchString(tok) {
				return nil, fail(422, "invalid_push_token", "not an FCM registration token")
			}
		default:
			return nil, fail(422, "not_an_app", "following a request is only for the iOS and Android apps")
		}
		server := strings.ToLower(strings.TrimSpace(in.Body.Server))
		if server != "" && !serverHostRe.MatchString(server) {
			return nil, fail(422, "invalid_server", "not a host name")
		}
		act := strings.TrimSpace(in.Body.ActivityID)
		if act != "" && !activityIDRe.MatchString(act) {
			return nil, fail(422, "invalid_activity_id", "not an activity id")
		}
		if sess.Platform != "ios" {
			act = ""
		}
		if d.Demo || d.Follow == nil || d.Push == nil || !d.Push.Configured(ctx) {
			return nil, fail(409, "push_unavailable", "push notifications are not set up on this server")
		}
		r, err := d.Store.GetRequest(ctx, in.ID)
		if err != nil {
			return nil, reqErr(err)
		}
		v, err := d.Requests.Get(ctx, u, in.ID)
		if err != nil {
			return nil, reqErr(err)
		}
		stage, percent, eta := d.Requests.LiveState(r)
		if follow.Final(stage) {
			return nil, fail(409, "follow_final", "this request is finished")
		}
		mine, err := d.Follow.List(ctx, sess.ID)
		if err != nil {
			return nil, err
		}
		if len(mine) >= follow.MaxPerSession && !slices.ContainsFunc(mine, func(f store.Follow) bool { return f.RequestID == v.ID }) {
			return nil, fail(409, "follow_limit", "too many requests are followed on this device")
		}
		f := &store.Follow{RequestID: v.ID, UserID: u.ID, SessionID: sess.ID, Platform: sess.Platform, PushToken: tok,
			Sandbox: in.Body.Sandbox && sess.Platform == "ios", Server: server, ActivityID: act}
		if err := d.Follow.Put(ctx, f, stage, percent); err != nil {
			return nil, err
		}
		return &struct{ Body requestFollow }{requestFollow{RequestID: v.ID, Platform: f.Platform, Stage: stage, Percent: percent,
			ETASeconds: eta, UpdatedAt: f.LastSentAt, ExpiresAt: f.ExpiresAt}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "unfollowRequest", Method: http.MethodDelete, Path: "/requests/{id}/follow",
		Summary: "Stop following a request from this app", Tags: []string{"requests"}, Security: sec, DefaultStatus: http.StatusNoContent,
		Description: "Sends nothing: the app ends its Live Activity or notification itself.",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, in *followIDInput) (*struct{}, error) {
		if _, err := requireUser(ctx); err != nil {
			return nil, err
		}
		if sess := sessionFrom(ctx); sess != nil && d.Follow != nil {
			return nil, d.Follow.Delete(ctx, sess.ID, in.ID)
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listFollows", Method: http.MethodGet, Path: "/me/follows",
		Summary: "Requests followed from this app", Tags: []string{"push"}, Security: sec,
		Description: "Lets the app match its Live Activities or notifications with the server at launch.",
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body followList }, error) {
		if _, err := requireUser(ctx); err != nil {
			return nil, err
		}
		out := followList{Items: []requestFollow{}}
		sess := sessionFrom(ctx)
		if sess == nil || d.Follow == nil {
			return &struct{ Body followList }{out}, nil
		}
		rows, err := d.Follow.List(ctx, sess.ID)
		if err != nil {
			return nil, err
		}
		for _, f := range rows {
			out.Items = append(out.Items, requestFollow{RequestID: f.RequestID, Platform: f.Platform, Stage: f.LastStage,
				Percent: f.LastPercent, UpdatedAt: f.LastSentAt, ExpiresAt: f.ExpiresAt})
		}
		return &struct{ Body followList }{out}, nil
	})
}
