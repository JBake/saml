// Package samlsp provides helpers that can be used to protect web services using SAML.
package samlsp

import (
	"crypto/rsa"
	"crypto/x509"
	"net/http"
	"net/url"

	dsig "github.com/russellhaering/goxmldsig"

	"github.com/JBake/saml"
)

// Options represents the parameters for creating a new middleware
type Options struct {
	EntityID          string
	URL               url.URL
	Key               *rsa.PrivateKey
	Certificate       *x509.Certificate
	Intermediates     []*x509.Certificate
	AllowIDPInitiated bool
	IDPMetadata       *saml.EntityDescriptor
	SignRequest       bool
	ForceAuthn        bool // TODO(ross): this should be *bool
	CookieSameSite    http.SameSite
	RelayStateFunc    func(w http.ResponseWriter, r *http.Request) string
}

// DefaultSessionCodec returns the default SessionCodec for the provided options,
// a JWTSessionCodec configured to issue signed tokens.
func DefaultSessionCodec(opts Options) JWTSessionCodec {
	return JWTSessionCodec{
		SigningMethod: defaultJWTSigningMethod,
		Audience:      opts.URL.String(),
		Issuer:        opts.URL.String(),
		MaxAge:        defaultSessionMaxAge,
		Key:           opts.Key,
	}
}

// DefaultSessionProvider returns the default SessionProvider for the provided options,
// a CookieSessionProvider configured to store sessions in a cookie.
func DefaultSessionProvider(opts Options) CookieSessionProvider {
	return CookieSessionProvider{
		Name:     defaultSessionCookieName,
		Domain:   opts.URL.Host,
		MaxAge:   defaultSessionMaxAge,
		HTTPOnly: true,
		Secure:   opts.URL.Scheme == "https",
		SameSite: opts.CookieSameSite,
		Codec:    DefaultSessionCodec(opts),
	}
}

// DefaultTrackedRequestCodec returns a new TrackedRequestCodec for the provided
// options, a JWTTrackedRequestCodec that uses a JWT to encode TrackedRequests.
func DefaultTrackedRequestCodec(opts Options) JWTTrackedRequestCodec {
	return JWTTrackedRequestCodec{
		SigningMethod: defaultJWTSigningMethod,
		Audience:      opts.URL.String(),
		Issuer:        opts.URL.String(),
		MaxAge:        saml.MaxIssueDelay,
		Key:           opts.Key,
	}
}

// DefaultRequestTracker returns a new RequestTracker for the provided options,
// a CookieRequestTracker which uses cookies to track pending requests.
func DefaultRequestTracker(opts Options, serviceProvider *saml.ServiceProvider) CookieRequestTracker {
	return CookieRequestTracker{
		ServiceProvider: serviceProvider,
		NamePrefix:      "saml_",
		Codec:           DefaultTrackedRequestCodec(opts),
		MaxAge:          saml.MaxIssueDelay,
		RelayStateFunc:  opts.RelayStateFunc,
		SameSite:        opts.CookieSameSite,
	}
}

// todo: move this up by the other options

type ManualOptions struct {
	EntityID          string
	Key               *rsa.PrivateKey
	Certificate       *x509.Certificate
	Intermediates     []*x509.Certificate
	AllowIDPInitiated bool
	IDPMetadata       *saml.EntityDescriptor
	SignRequest       bool
	ForceAuthn        bool // TODO(ross): this should be *bool
	MetadataURL       url.URL
	AcsURL            url.URL
	SloURL            url.URL
}

// ManagedServiceProvider returns a saml.ServiceProvider managed outside of middleware
func ManagedServiceProvider(opts ManualOptions) saml.ServiceProvider {
	metadataURL := opts.MetadataURL
	acsURL := opts.AcsURL
	sloURL := opts.SloURL

	var forceAuthn *bool
	if opts.ForceAuthn {
		forceAuthn = &opts.ForceAuthn
	}
	signatureMethod := dsig.RSASHA1SignatureMethod
	if !opts.SignRequest {
		signatureMethod = ""
	}

	return saml.ServiceProvider{
		EntityID:          opts.EntityID,
		Key:               opts.Key,
		Certificate:       opts.Certificate,
		Intermediates:     opts.Intermediates,
		MetadataURL:       metadataURL,
		AcsURL:            acsURL,
		SloURL:            sloURL,
		IDPMetadata:       opts.IDPMetadata,
		ForceAuthn:        forceAuthn,
		SignatureMethod:   signatureMethod,
		AllowIDPInitiated: opts.AllowIDPInitiated,
	}
}

// DefaultServiceProvider returns the default saml.ServiceProvider for the provided
// options.
func DefaultServiceProvider(opts Options) saml.ServiceProvider {
	metadataURL := opts.URL.ResolveReference(&url.URL{Path: "saml/metadata"})
	acsURL := opts.URL.ResolveReference(&url.URL{Path: "saml/acs"})
	sloURL := opts.URL.ResolveReference(&url.URL{Path: "saml/slo"})

	var forceAuthn *bool
	if opts.ForceAuthn {
		forceAuthn = &opts.ForceAuthn
	}
	signatureMethod := dsig.RSASHA1SignatureMethod
	if !opts.SignRequest {
		signatureMethod = ""
	}

	return saml.ServiceProvider{
		EntityID:          opts.EntityID,
		Key:               opts.Key,
		Certificate:       opts.Certificate,
		Intermediates:     opts.Intermediates,
		MetadataURL:       *metadataURL,
		AcsURL:            *acsURL,
		SloURL:            *sloURL,
		IDPMetadata:       opts.IDPMetadata,
		ForceAuthn:        forceAuthn,
		SignatureMethod:   signatureMethod,
		AllowIDPInitiated: opts.AllowIDPInitiated,
	}
}

// New creates a new Middleware with the default providers for the
// given options.
//
// You can customize the behavior of the middleware in more detail by
// replacing and/or changing Session, RequestTracker, and ServiceProvider
// in the returned Middleware.
func New(opts Options) (*Middleware, error) {
	m := &Middleware{
		ServiceProvider: DefaultServiceProvider(opts),
		Binding:         "",
		OnError:         DefaultOnError,
		Session:         DefaultSessionProvider(opts),
	}
	m.RequestTracker = DefaultRequestTracker(opts, &m.ServiceProvider)

	return m, nil
}
