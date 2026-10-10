package api

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

var (
	apnsTokenRe = regexp.MustCompile(`^[0-9a-fA-F]{64,200}$`)
	fcmTokenRe  = regexp.MustCompile(`^[A-Za-z0-9:_-]{20,1000}$`)
)

// pushDevice is what the API shows about this app's push registration. The token is never returned.
type pushDevice struct {
	Platform   string `json:"platform" enum:"ios,android"`
	Sandbox    bool   `json:"sandbox"`
	Language   string `json:"language"`
	Categories int    `json:"categories" doc:"Bitmask: 1 my requests, 2 new requests to approve (admins), 4 issues"`
	UpdatedAt  int64  `json:"updatedAt"`
}

func toPushDevice(d *store.Device) pushDevice {
	return pushDevice{Platform: d.Platform, Sandbox: d.Sandbox, Language: d.Language, Categories: d.Categories, UpdatedAt: d.UpdatedAt}
}

type registerDeviceInput struct {
	Body struct {
		PushToken  string `json:"pushToken" minLength:"20" maxLength:"1024" doc:"APNs device token (hex) or FCM registration token"`
		Sandbox    bool   `json:"sandbox,omitempty" doc:"iOS only: the token comes from a development build (APNs sandbox)"`
		Language   string `json:"language,omitempty" maxLength:"16" doc:"Language of the alert text (en or fr); default en"`
		Categories *int   `json:"categories,omitempty" minimum:"0" maximum:"7" doc:"Bitmask of the notifications wanted; default all (7), unchanged on refresh when omitted"`
	}
}

func registerDevices(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}, {"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "registerDevice", Method: http.MethodPut, Path: "/me/devices/current",
		Summary: "Register or refresh this app's push token", Tags: []string{"push"}, Security: sec,
		Description: "Call it at every launch and whenever the OS hands out a new token or the user changes the notification toggles. The registration belongs to the current session and disappears when it ends.",
		Errors:      []int{http.StatusUnauthorized, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *registerDeviceInput) (*struct{ Body pushDevice }, error) {
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
			if !apnsTokenRe.MatchString(tok) {
				return nil, fail(422, "invalid_push_token", "not an APNs device token")
			}
		case "android":
			if !fcmTokenRe.MatchString(tok) {
				return nil, fail(422, "invalid_push_token", "not an FCM registration token")
			}
		default:
			return nil, fail(422, "not_an_app", "push is only for the iOS and Android apps")
		}
		lang := strings.ToLower(strings.TrimSpace(in.Body.Language))
		if len(lang) > 2 {
			lang = lang[:2]
		}
		old, oldErr := d.Store.DeviceBySession(ctx, sess.ID)
		if lang == "" && oldErr == nil {
			lang = old.Language // a refresh that omits it keeps what was saved
		}
		if lang == "" {
			lang = "en"
		}
		dev := &store.Device{
			UserID: u.ID, SessionID: sess.ID, Platform: sess.Platform, PushToken: tok,
			Sandbox: in.Body.Sandbox && sess.Platform == "ios", AppVersion: sess.AppVersion, Language: lang,
			Categories: store.PushAll,
		}
		if in.Body.Categories != nil {
			dev.Categories = *in.Body.Categories
		} else if oldErr == nil {
			dev.Categories = old.Categories
		}
		if err := d.Store.SaveDevice(ctx, dev); err != nil {
			return nil, err
		}
		return &struct{ Body pushDevice }{toPushDevice(dev)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getDevice", Method: http.MethodGet, Path: "/me/devices/current",
		Summary: "This app's push registration", Tags: []string{"push"}, Security: sec,
		Errors: []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body pushDevice }, error) {
		if _, err := requireUser(ctx); err != nil {
			return nil, err
		}
		sess := sessionFrom(ctx)
		if sess == nil {
			return nil, fail(404, "not_found", "not registered")
		}
		dev, err := d.Store.DeviceBySession(ctx, sess.ID)
		if errors.Is(err, store.ErrNotFound) {
			return nil, fail(404, "not_found", "not registered")
		}
		if err != nil {
			return nil, err
		}
		return &struct{ Body pushDevice }{toPushDevice(dev)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "unregisterDevice", Method: http.MethodDelete, Path: "/me/devices/current",
		Summary: "Stop push notifications on this device", Tags: []string{"push"}, Security: sec, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if _, err := requireUser(ctx); err != nil {
			return nil, err
		}
		if sess := sessionFrom(ctx); sess != nil {
			return nil, d.Store.DeleteDeviceBySession(ctx, sess.ID)
		}
		return nil, nil
	})
}
