package handler

import (
	"net/http"

	"github.com/antihax/goesi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type handler struct {
	done         chan<- struct{}
	log          handlerLogger
	tokenStorage tokenStorage
	esi          *goesi.APIClient
	sso          *goesi.SSOAuthenticator
	router       http.Handler
	store        *sessions.CookieStore
	scopes       []string
}

type handlerLogger interface {
	Infow(string, ...interface{})
	Errorw(string, ...interface{})
}

type tokenStorage interface {
	Write(oauth2.Token) error
}

// New constructs new API http handler.
func New(
	done chan<- struct{},
	log handlerLogger,
	client *http.Client,
	userAgent string,
	tokenStorage tokenStorage,
	secretKey []byte,
	clientID, ssoSecret string,
	callbackURL string,
	scopes []string,
) http.Handler {
	esi := goesi.NewAPIClient(client, userAgent)
	sso := goesi.NewSSOAuthenticatorV2(client, clientID, ssoSecret, callbackURL, scopes)
	r := chi.NewRouter()
	h := handler{
		done:         done,
		log:          log,
		tokenStorage: tokenStorage,
		esi:          esi,
		sso:          sso,
		router:       r,
		store:        sessions.NewCookieStore(secretKey),
		scopes:       scopes,
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", ErrorHandler(h.indexHandler, h.log))
	r.Get("/login", ErrorHandler(h.loginHandler, h.log))
	r.Get("/callback", ErrorHandler(h.callbackHandler, h.log))
	return &h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *handler) session(r *http.Request) *sessions.Session {
	sess, _ := h.store.Get(r, "eve-bot-session")
	return sess
}

func (h *handler) tokenSource(r *http.Request, w http.ResponseWriter) (oauth2.TokenSource, error) {
	session := h.session(r)
	token, ok := session.Values["token"].(oauth2.Token)
	if !ok {
		return nil, errors.Errorf("no token saved in session")
	}

	ts := h.sso.TokenSource(&token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, errors.Wrapf(err, "error getting token")
	}

	if token != *newToken {
		// Save token.
		session.Values["token"] = *newToken
		err = session.Save(r, w)
		if err != nil {
			return nil, errors.Wrap(err, "unable to save session")
		}
	}

	return ts, nil
}

func (h *handler) character(r *http.Request) (*goesi.VerifyResponse, error) {
	session := h.session(r)
	char, ok := session.Values["character"].(goesi.VerifyResponse)
	if !ok {
		return nil, errors.New("unable to get character from session")
	}
	return &char, nil
}
