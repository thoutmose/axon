package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"

	"github.com/thoutmose/axon/api/internal/audit"
	"github.com/thoutmose/axon/api/internal/httpx"
)

func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	var version time.Time
	var out struct {
		ID                     uuid.UUID  `json:"id"`
		DisplayName            string     `json:"display_name"`
		Timezone               string     `json:"timezone"`
		CreatedAt              time.Time  `json:"created_at"`
		TOTPEnabled            bool       `json:"totp_enabled"`
		Passkeys               int        `json:"passkeys"`
		RecoveryCodesRemaining int        `json:"recovery_codes_remaining"`
		Demo                   bool       `json:"demo"`
		AvatarUpdatedAt        *time.Time `json:"avatar_updated_at"`
	}
	err := s.db.QueryRow(r.Context(),
		`SELECT u.updated_at, u.id, u.display_name, u.timezone, u.created_at, u.totp_enabled_at IS NOT NULL,
		        (SELECT count(*) FROM passkeys WHERE user_id = u.id),
		        (SELECT count(*) FROM recovery_codes WHERE user_id = u.id AND used_at IS NULL), u.demo, u.avatar_updated_at
		 FROM users u WHERE u.id = $1`, id.UserID).
		Scan(&version, &out.ID, &out.DisplayName, &out.Timezone, &out.CreatedAt, &out.TOTPEnabled, &out.Passkeys, &out.RecoveryCodesRemaining, &out.Demo, &out.AvatarUpdatedAt)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("ETag", httpx.ETag(version))
	httpx.JSON(w, http.StatusOK, out)
}

// updateMe changes the account's name and/or time zone. It needs If-Match
// with the ETag from GET /v1/me, so a stale tab cannot overwrite a newer change.
func (s *Service) updateMe(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	version, ok := httpx.IfMatch(w, r)
	if !ok {
		return
	}
	var body struct {
		DisplayName *string `json:"display_name"`
		Timezone    *string `json:"timezone"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var fields []string
	if body.DisplayName != nil {
		name, valid := cleanName(*body.DisplayName)
		if !valid {
			httpx.Problem(w, http.StatusUnprocessableEntity, "Enter a name between 1 and 100 characters.")
			return
		}
		body.DisplayName = &name
		fields = append(fields, "display_name")
	}
	if body.Timezone != nil {
		if !validZone(*body.Timezone) {
			httpx.Problem(w, http.StatusUnprocessableEntity, "Use an IANA time zone such as Europe/Paris.")
			return
		}
		fields = append(fields, "timezone")
	}
	errStale := errors.New("stale")
	var updated time.Time
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(r.Context(),
			`UPDATE users SET display_name = coalesce($3, display_name), timezone = coalesce($4, timezone),
			 updated_at = clock_timestamp() WHERE id = $1 AND updated_at = $2 RETURNING updated_at`,
			id.UserID, version, body.DisplayName, body.Timezone).Scan(&updated)
		if errors.Is(err, pgx.ErrNoRows) {
			return errStale
		}
		if err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.AccountUpdated, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"fields": fields})
	})
	switch {
	case errors.Is(err, errStale):
		httpx.Problem(w, http.StatusPreconditionFailed, "Your account changed since you loaded it. Reload and try again.")
	case err != nil:
		s.fail(w, r, err)
	default:
		w.Header().Set("ETag", httpx.ETag(updated))
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Sessions ---

func (s *Service) logoutAll(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id = $1`, id.UserID); err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.SessionsRevokedAll, httpx.ClientIP(r), r.UserAgent(), nil)
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.clearCookie(w, sessionCookie)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) listSessions(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	type session struct {
		ID         uuid.UUID `json:"id"`
		UserAgent  string    `json:"user_agent"`
		CreatedAt  time.Time `json:"created_at"`
		LastSeenAt time.Time `json:"last_seen_at"`
		Current    bool      `json:"current"`
	}
	rows, err := s.db.Query(r.Context(),
		`SELECT id, user_agent, created_at, last_seen_at FROM sessions
		 WHERE user_id = $1 AND expires_at > now() ORDER BY last_seen_at DESC LIMIT 100`, id.UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (session, error) {
		var x session
		err := row.Scan(&x.ID, &x.UserAgent, &x.CreatedAt, &x.LastSeenAt)
		x.Current = x.ID == id.SessionID
		return x, err
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Service) revokeSession(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	target, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such session.")
		return
	}
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(), `DELETE FROM sessions WHERE id = $1 AND user_id = $2`, target, id.UserID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errNotFound
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.SessionRevoked, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"session_id": target})
	})
	switch {
	case errors.Is(err, errNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such session.")
	case err != nil:
		s.fail(w, r, err)
	default:
		if target == id.SessionID {
			s.clearCookie(w, sessionCookie)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Passkeys ---

func (s *Service) listPasskeys(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	type passkey struct {
		ID         uuid.UUID  `json:"id"`
		Name       string     `json:"name"`
		CreatedAt  time.Time  `json:"created_at"`
		LastUsedAt *time.Time `json:"last_used_at"`
	}
	rows, err := s.db.Query(r.Context(),
		`SELECT id, name, created_at, last_used_at FROM passkeys WHERE user_id = $1 ORDER BY created_at`, id.UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[passkey])
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Service) addPasskeyBegin(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	name, ok := cleanName(body.Name)
	if !ok {
		httpx.Problem(w, http.StatusUnprocessableEntity, "Enter a passkey name between 1 and 100 characters.")
		return
	}
	u, err := loadUser(r.Context(), s.db, id.UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	options, session, err := s.wa.BeginRegistration(u, creationOptions(u.credentials)...)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.saveCeremony(r.Context(), w, "add_passkey", ceremony{UserID: &id.UserID, DisplayName: name, Session: *session}); err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, options)
}

func (s *Service) addPasskeyFinish(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		c, err := s.takeCeremony(r.Context(), tx, w, r, "add_passkey")
		if err != nil {
			return err
		}
		if c.UserID == nil || *c.UserID != id.UserID {
			return errNotFound
		}
		u, err := loadUser(r.Context(), tx, id.UserID)
		if err != nil {
			return err
		}
		cred, err := s.wa.FinishRegistration(u, c.Session, r)
		if err != nil {
			return errors.Join(errCeremonyFailed, err)
		}
		if err := insertPasskey(r, tx, id.UserID, c.DisplayName, cred); err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.PasskeyAdded, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"name": c.DisplayName})
	})
	switch {
	case errors.Is(err, errNotFound), errors.Is(err, errCeremonyFailed):
		s.log.Info("add-passkey ceremony rejected", "err", err)
		httpx.Problem(w, http.StatusBadRequest, "The passkey could not be added. Start again.")
	case err != nil:
		s.fail(w, r, err)
	default:
		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Service) removePasskey(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	target, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such passkey.")
		return
	}
	errLast := errors.New("last passkey")
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var name string
		var total int
		err := tx.QueryRow(r.Context(),
			`SELECT name, (SELECT count(*) FROM passkeys WHERE user_id = $2) FROM passkeys WHERE id = $1 AND user_id = $2 FOR UPDATE`,
			target, id.UserID).Scan(&name, &total)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNotFound
		}
		if err != nil {
			return err
		}
		if total <= 1 {
			return errLast
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM passkeys WHERE id = $1`, target); err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.PasskeyRemoved, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"name": name})
	})
	switch {
	case errors.Is(err, errNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such passkey.")
	case errors.Is(err, errLast):
		httpx.Problem(w, http.StatusConflict, "Add another passkey before removing your last one.")
	case err != nil:
		s.fail(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Service) regenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	var codes []string
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var err error
		if codes, err = issueRecoveryCodes(r.Context(), tx, id.UserID); err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, audit.RecoveryCodesIssued, httpx.ClientIP(r), r.UserAgent(), nil)
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

// --- TOTP ---

// useTOTP accepts a code for the current, previous, or next 30-second step
// and records the step, so the same code cannot be used twice. Call it inside
// the transaction that acts on the code, after locking the user's row.
func (s *Service) useTOTP(ctx context.Context, tx pgx.Tx, userID uuid.UUID, sealed []byte, code string) bool {
	if s.sealer == nil || sealed == nil || len(code) != 6 {
		return false
	}
	secret, err := s.sealer.Open(sealed)
	if err != nil {
		s.log.Error("cannot open TOTP secret; was AXON_ENCRYPTION_KEY changed?", "err", err)
		return false
	}
	now := time.Now()
	for _, skew := range []int64{0, -1, 1} {
		at := now.Add(time.Duration(skew) * 30 * time.Second)
		want, err := totp.GenerateCode(string(secret), at)
		if err != nil || subtle.ConstantTimeCompare([]byte(want), []byte(code)) != 1 {
			continue
		}
		step := at.Unix() / 30
		tag, err := tx.Exec(ctx, `UPDATE users SET totp_last_step = $2 WHERE id = $1 AND totp_last_step < $2`, userID, step)
		return err == nil && tag.RowsAffected() == 1
	}
	return false
}

func (s *Service) totpBegin(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	if s.sealer == nil {
		httpx.Problem(w, http.StatusServiceUnavailable, "Authenticator codes are not configured on this Axon.")
		return
	}
	var name string
	var enabled bool
	err := s.db.QueryRow(r.Context(), `SELECT display_name, totp_enabled_at IS NOT NULL FROM users WHERE id = $1`, id.UserID).
		Scan(&name, &enabled)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if enabled {
		httpx.Problem(w, http.StatusConflict, "Authenticator codes are already on. Turn them off first to change the authenticator.")
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "Axon", AccountName: name})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// Stored but inactive until confirmed with a valid code.
	if _, err := s.db.Exec(r.Context(), `UPDATE users SET totp_secret = $1 WHERE id = $2`,
		s.sealer.Seal([]byte(key.Secret())), id.UserID); err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"secret": key.Secret(), "otpauth_url": key.URL()})
}

func (s *Service) totpConfirm(w http.ResponseWriter, r *http.Request) {
	s.totpSwitch(w, r, true)
}

func (s *Service) totpDisable(w http.ResponseWriter, r *http.Request) {
	s.totpSwitch(w, r, false)
}

// totpSwitch turns TOTP on (confirming a pending secret) or off; both need a valid current code.
func (s *Service) totpSwitch(w http.ResponseWriter, r *http.Request, on bool) {
	id, _ := FromContext(r.Context())
	var body struct {
		Code string `json:"code"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	errBadCode := errors.New("bad code")
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var sealed []byte
		var enabled bool
		err := tx.QueryRow(r.Context(), `SELECT totp_secret, totp_enabled_at IS NOT NULL FROM users WHERE id = $1 FOR UPDATE`, id.UserID).
			Scan(&sealed, &enabled)
		if err != nil {
			return err
		}
		if enabled == on || !s.useTOTP(r.Context(), tx, id.UserID, sealed, body.Code) {
			return errBadCode
		}
		action := audit.TOTPEnabled
		query := `UPDATE users SET totp_enabled_at = now() WHERE id = $1`
		if !on {
			action = audit.TOTPDisabled
			query = `UPDATE users SET totp_enabled_at = NULL, totp_secret = NULL WHERE id = $1`
		}
		if _, err := tx.Exec(r.Context(), query, id.UserID); err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, id.UserID, action, httpx.ClientIP(r), r.UserAgent(), nil)
	})
	switch {
	case errors.Is(err, errBadCode):
		httpx.Problem(w, http.StatusUnprocessableEntity, "That code is not valid. Check your authenticator app and try again.")
	case err != nil:
		s.fail(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Service) listAudit(w http.ResponseWriter, r *http.Request) {
	id, _ := FromContext(r.Context())
	events, err := audit.List(r.Context(), s.db, id.UserID, 200)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": events})
}

// validZone says whether tz is an IANA time zone this server knows.
func validZone(tz string) bool {
	if tz == "" || tz == "Local" || len(tz) > 64 {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}
