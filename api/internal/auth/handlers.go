package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/audit"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Routes mounts the authentication and account endpoints.
func (s *Service) Routes(r chi.Router) {
	r.Get("/v1/auth/status", s.status)
	r.Get("/v1/auth/providers", s.providersList)
	r.Post("/v1/auth/logout", s.logout)
	r.Group(func(r chi.Router) {
		r.Use(s.signInLimit.middleware)
		r.Post("/v1/auth/register/begin", s.registerBegin)
		r.Post("/v1/auth/register/finish", s.registerFinish)
		r.Post("/v1/auth/login/begin", s.loginBegin)
		r.Post("/v1/auth/login/finish", s.loginFinish)
		r.Post("/v1/auth/recovery", s.recoveryLogin)
		r.Post("/v1/auth/demo", s.demoLogin)
	})
	// Each start stores a state row: limited on its own, more loosely than
	// passkey attempts. The callback needs that state's cookie.
	r.With(s.oauthLimit.middleware).Get("/v1/auth/oauth/{provider}/start", s.oauthStart)
	r.Get("/v1/auth/oauth/{provider}/callback", s.oauthCallback)
	r.Group(func(r chi.Router) {
		r.Use(s.RequireSession)
		r.Get("/v1/me", s.me)
		r.Patch("/v1/me", s.updateMe)
		r.Get("/v1/me/tokens", s.listTokens)
		r.With(NotDemo).Post("/v1/me/tokens", s.createToken)
		r.Delete("/v1/me/tokens/{id}", s.revokeToken)
		r.Post("/v1/me/logout-all", s.logoutAll)
		r.Get("/v1/me/sessions", s.listSessions)
		r.Delete("/v1/me/sessions/{id}", s.revokeSession)
		r.Get("/v1/me/passkeys", s.listPasskeys)
		r.With(NotDemo).Post("/v1/me/passkeys/begin", s.addPasskeyBegin)
		r.With(NotDemo).Post("/v1/me/passkeys/finish", s.addPasskeyFinish)
		r.With(NotDemo).Delete("/v1/me/passkeys/{id}", s.removePasskey)
		r.With(NotDemo).Post("/v1/me/recovery-codes", s.regenerateRecoveryCodes)
		r.With(NotDemo).Post("/v1/me/totp/begin", s.totpBegin)
		r.With(NotDemo).Post("/v1/me/totp/confirm", s.totpConfirm)
		r.With(NotDemo).Post("/v1/me/totp/disable", s.totpDisable)
		r.Get("/v1/me/audit", s.listAudit)
		r.Get("/v1/me/identities", s.listIdentities)
		r.Delete("/v1/me/identities/{id}", s.unlinkIdentity)
	})
}

// fail logs an unexpected error and answers with a generic 500.
func (s *Service) fail(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "route", chi.RouteContext(r.Context()).RoutePattern(), "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

func cleanName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	return name, name != "" && len([]rune(name)) <= 100
}

// --- Registration and sign-in ---

func (s *Service) status(w http.ResponseWriter, r *http.Request) {
	open, err := s.registrationOpen(r.Context(), s.db)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	id, idErr := s.identify(r)
	httpx.JSON(w, http.StatusOK, map[string]bool{
		"registration_open": open,
		"signed_in":         idErr == nil,
		"totp_available":    s.sealer != nil,
		"demo_available":    s.demo,
		"demo_session":      idErr == nil && id.Demo,
	})
}

// demoSession is how long a demo sign-in lasts.
const demoSession = 12 * time.Hour

// demoLogin signs into the shared demo account, when the server offers one.
func (s *Service) demoLogin(w http.ResponseWriter, r *http.Request) {
	if !s.demo {
		httpx.Problem(w, http.StatusNotFound, "This Axon has no demo account.")
		return
	}
	var id uuid.UUID
	err := s.db.QueryRow(r.Context(), `SELECT id FROM users WHERE demo`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Problem(w, http.StatusServiceUnavailable, "The demo is being prepared. Try again in a minute.")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.startSessionFor(r.Context(), s.db, w, r, id, demoSession); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) registerBegin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DisplayName string `json:"display_name"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	name, ok := cleanName(body.DisplayName)
	if !ok {
		httpx.Problem(w, http.StatusUnprocessableEntity, "Enter a name between 1 and 100 characters.")
		return
	}
	open, err := s.registrationOpen(r.Context(), s.db)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !open {
		httpx.Problem(w, http.StatusForbidden, "Registration is closed on this Axon.")
		return
	}
	u := &user{ID: uuid.Must(uuid.NewV7()), DisplayName: name}
	options, session, err := s.wa.BeginRegistration(u, creationOptions(nil)...)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.saveCeremony(r.Context(), w, "register", ceremony{DisplayName: name, Session: *session}); err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, options)
}

func (s *Service) registerFinish(w http.ResponseWriter, r *http.Request) {
	var codes []string
	var displayName string
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		// Serialize registrations so "first account only" cannot race.
		if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(7364001)`); err != nil {
			return err
		}
		if open, err := s.registrationOpen(r.Context(), tx); err != nil || !open {
			return errors.Join(err, errRegistrationClosed)
		}
		c, err := s.takeCeremony(r.Context(), tx, w, r, "register")
		if err != nil {
			return err
		}
		userID, err := uuid.FromBytes(c.Session.UserID)
		if err != nil {
			return errNotFound
		}
		u := &user{ID: userID, DisplayName: c.DisplayName}
		cred, err := s.wa.FinishRegistration(u, c.Session, r)
		if err != nil {
			return errors.Join(errCeremonyFailed, err)
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO users (id, display_name) VALUES ($1, $2)`, u.ID, u.DisplayName); err != nil {
			return err
		}
		if err := insertPasskey(r, tx, u.ID, "First passkey", cred); err != nil {
			return err
		}
		if codes, err = issueRecoveryCodes(r.Context(), tx, u.ID); err != nil {
			return err
		}
		if err := audit.Record(r.Context(), tx, u.ID, audit.AccountCreated, httpx.ClientIP(r), r.UserAgent(), nil); err != nil {
			return err
		}
		displayName = u.DisplayName
		return s.startSession(r.Context(), tx, w, r, u.ID)
	})
	switch {
	case errors.Is(err, errRegistrationClosed):
		httpx.Problem(w, http.StatusForbidden, "Registration is closed on this Axon.")
	case errors.Is(err, errNotFound), errors.Is(err, errCeremonyFailed):
		s.log.Info("registration ceremony rejected", "err", err)
		httpx.Problem(w, http.StatusBadRequest, "The passkey could not be created. Start again.")
	case err != nil:
		s.fail(w, r, err)
	default:
		httpx.JSON(w, http.StatusCreated, map[string]any{"display_name": displayName, "recovery_codes": codes})
	}
}

var (
	errRegistrationClosed = errors.New("registration closed")
	errCeremonyFailed     = errors.New("webauthn ceremony failed")
)

func insertPasskey(r *http.Request, q db.Querier, userID uuid.UUID, name string, cred *webauthn.Credential) error {
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	_, err = q.Exec(r.Context(),
		`INSERT INTO passkeys (id, user_id, credential_id, credential, name) VALUES ($1, $2, $3, $4, $5)`,
		uuid.Must(uuid.NewV7()), userID, cred.ID, raw, name)
	return err
}

func (s *Service) loginBegin(w http.ResponseWriter, r *http.Request) {
	options, session, err := s.wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.saveCeremony(r.Context(), w, "login", ceremony{Session: *session}); err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, options)
}

func (s *Service) loginFinish(w http.ResponseWriter, r *http.Request) {
	var displayName string
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		c, err := s.takeCeremony(r.Context(), tx, w, r, "login")
		if err != nil {
			return err
		}
		var found *user
		handler := func(_, userHandle []byte) (webauthn.User, error) {
			id, err := uuid.FromBytes(userHandle)
			if err != nil {
				return nil, errNotFound
			}
			found, err = loadUser(r.Context(), tx, id)
			return found, err
		}
		_, cred, err := s.wa.FinishPasskeyLogin(handler, c.Session, r)
		if err != nil {
			return errors.Join(errCeremonyFailed, err)
		}
		if cred.Authenticator.CloneWarning {
			return errors.Join(errCeremonyFailed, errors.New("authenticator clone warning"))
		}
		raw, err := json.Marshal(cred)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(),
			`UPDATE passkeys SET credential = $1, last_used_at = now() WHERE credential_id = $2 AND user_id = $3`,
			raw, cred.ID, found.ID); err != nil {
			return err
		}
		if err := audit.Record(r.Context(), tx, found.ID, audit.SignInPasskey, httpx.ClientIP(r), r.UserAgent(), nil); err != nil {
			return err
		}
		displayName = found.DisplayName
		return s.startSession(r.Context(), tx, w, r, found.ID)
	})
	switch {
	case errors.Is(err, errNotFound), errors.Is(err, errCeremonyFailed):
		s.log.Info("sign-in ceremony rejected", "err", err)
		httpx.Problem(w, http.StatusUnauthorized, "Sign-in failed. Try again with a passkey for this Axon.")
	case err != nil:
		s.fail(w, r, err)
	default:
		httpx.JSON(w, http.StatusOK, map[string]string{"display_name": displayName})
	}
}

// Recovery throttling: five wrong authenticator codes for an account lock
// recovery sign-in for 15 minutes, on top of the per-IP sign-in limit.
const (
	recoveryMaxFailures = 5
	recoveryLock        = 15 * time.Minute
)

func (s *Service) recoveryLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
		TOTP string `json:"totp"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var displayName string
	var failedUser *uuid.UUID
	errInvalid := errors.New("invalid recovery attempt")
	errLocked := errors.New("recovery locked")
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var codeID, userID uuid.UUID
		var sealed []byte
		var totpOn, lockedUntil *time.Time
		err := tx.QueryRow(r.Context(),
			`SELECT c.id, u.id, u.display_name, u.totp_secret, u.totp_enabled_at, u.recovery_locked_until
			 FROM recovery_codes c JOIN users u ON u.id = c.user_id
			 WHERE c.code_hash = $1 AND c.used_at IS NULL FOR UPDATE`,
			hashRecoveryCode(body.Code)).Scan(&codeID, &userID, &displayName, &sealed, &totpOn, &lockedUntil)
		if errors.Is(err, pgx.ErrNoRows) {
			return errInvalid
		}
		if err != nil {
			return err
		}
		if lockedUntil != nil && lockedUntil.After(time.Now()) {
			return errLocked
		}
		if totpOn != nil && !s.useTOTP(r.Context(), tx, userID, sealed, body.TOTP) {
			failedUser = &userID
			return errInvalid
		}
		if _, err := tx.Exec(r.Context(), `UPDATE recovery_codes SET used_at = now() WHERE id = $1`, codeID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `UPDATE users SET recovery_failures = 0, recovery_locked_until = NULL WHERE id = $1`, userID); err != nil {
			return err
		}
		if err := audit.Record(r.Context(), tx, userID, audit.SignInRecoveryCode, httpx.ClientIP(r), r.UserAgent(), nil); err != nil {
			return err
		}
		return s.startSession(r.Context(), tx, w, r, userID)
	})
	if failedUser != nil {
		// Counted outside the rolled-back transaction, so failures add up; at
		// the limit the lock starts and the count begins again.
		if uerr := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
			if _, err := tx.Exec(r.Context(), `UPDATE users SET
				recovery_locked_until = CASE WHEN recovery_failures + 1 >= $2 THEN now() + make_interval(secs => $3) ELSE recovery_locked_until END,
				recovery_failures = CASE WHEN recovery_failures + 1 >= $2 THEN 0 ELSE recovery_failures + 1 END
				WHERE id = $1`, *failedUser, recoveryMaxFailures, recoveryLock.Seconds()); err != nil {
				return err
			}
			return audit.Record(r.Context(), tx, *failedUser, audit.RecoveryFailed, httpx.ClientIP(r), r.UserAgent(), nil)
		}); uerr != nil {
			s.log.Error("recording a failed recovery attempt", "err", uerr)
		}
	}
	switch {
	case errors.Is(err, errLocked):
		w.Header().Set("Retry-After", "900")
		httpx.Problem(w, http.StatusTooManyRequests, "Too many wrong authenticator codes. Recovery sign-in is paused for 15 minutes.")
	case errors.Is(err, errInvalid):
		httpx.Problem(w, http.StatusUnauthorized, "That recovery code (or authenticator code) is not valid.")
	case err != nil:
		s.fail(w, r, err)
	default:
		httpx.JSON(w, http.StatusOK, map[string]string{"display_name": displayName})
	}
}

func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	if id, err := s.identify(r); err == nil {
		err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
			if _, err := tx.Exec(r.Context(), `DELETE FROM sessions WHERE id = $1`, id.SessionID); err != nil {
				return err
			}
			return audit.Record(r.Context(), tx, id.UserID, audit.SignOut, httpx.ClientIP(r), r.UserAgent(), nil)
		})
		if err != nil {
			s.fail(w, r, err)
			return
		}
	}
	s.clearCookie(w, sessionCookie)
	w.WriteHeader(http.StatusNoContent)
}
