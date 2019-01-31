package logout

import (
	"net/http"

	"github.com/RangelReale/osin"
	"github.com/golang/glog"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/openshift/origin/pkg/oauthserver"
	"github.com/openshift/origin/pkg/oauthserver/server/headers"
	"github.com/openshift/origin/pkg/oauthserver/server/redirect"
	"github.com/openshift/origin/pkg/oauthserver/server/session"
	"github.com/openshift/origin/pkg/oauthserver/server/tokenrequest"
)

const thenParam = "then"

func NewLogout(invalidator session.SessionInvalidator, redirect string) tokenrequest.Endpoints {
	return &logout{
		invalidator: invalidator,
		redirect:    redirect,
	}
}

type logout struct {
	invalidator session.SessionInvalidator
	redirect    string
}

func (l *logout) Install(mux oauthserver.Mux, paths ...string) {
	for _, path := range paths {
		mux.Handle(path, l)
	}
}

func (l *logout) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// TODO this seems like something that should happen automatically at a higher level
	// we also do not set these headers on the OAuth endpoints or the token request endpoint...
	headers.SetStandardHeaders(w)

	// TODO while having a POST provides some protection, this endpoint is invokable via JS.
	// we could easily add CSRF protection, but then it would make it really hard for the console
	// to actually use this endpoint.  we could have some alternative logout path that validates
	// the request based on the OAuth client secret, but all of that seems overkill for logout.
	// to make this perfectly safe, we would need the console to redirect to this page and then
	// have the user click logout.  forgo that for now to keep the UX of kube:admin clean.
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// invalidate with empty user to force session removal
	if err := l.invalidator.InvalidateAuthentication(w, &user.DefaultInfo{}); err != nil {
		glog.V(5).Infof("error logging out: %v", err)
		http.Error(w, "failed to log out", http.StatusInternalServerError)
		return
	}

	// optionally redirect if safe to do so
	if then := req.FormValue(thenParam); l.isValidRedirect(then) {
		http.Redirect(w, req, then, http.StatusFound)
		return
	}
}

func (l *logout) isValidRedirect(then string) bool {
	if redirect.IsServerRelativeURL(then) {
		return true
	}

	return osin.ValidateUri(l.redirect, then) == nil
}
