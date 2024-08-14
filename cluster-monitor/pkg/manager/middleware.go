// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/auth"

	"github.com/couchbase/tools-common/restutil"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"gopkg.in/square/go-jose.v2/jwt"
)

var unauthedEndpoints = []string{
	"/api/v1/self",
	"/api/v1/self/token",
	"/",
}

// authMiddleware will check the headers for auth information. Current supported systems are Basic and JWT Bearer token.
func (m *Manager) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// the /self endpoint is exempt of auth constraints
		if slices.Contains(unauthedEndpoints, request.URL.Path) || strings.HasPrefix(request.URL.Path, PathUIRoot) {
			next.ServeHTTP(writer, request)
			return
		}

		authParts := strings.SplitN(request.Header.Get("Authorization"), " ", 2)
		// invalid auth
		if len(authParts) != 2 {
			sendUnauthorized(writer, true)
			return
		}

		var err error
		switch authParts[0] {
		case "Basic":
			err = m.doBasicAuth(authParts[1])
		case "Bearer":
			err = m.doBearerJWTAuth(authParts[1])
		default:
			sendUnauthorized(writer, true)
			return
		}

		if err != nil {
			zap.S().Warnw("(Auth) Invalid user login attempt", "err", err)
			// If the user is logged into the UI with a JWT, no point in presenting a basic auth popup
			// the UI will catch the 401 and display its own prompt
			sendUnauthorized(writer, authParts[0] != "Bearer")
			return
		}

		// otherwise we are good to go
		next.ServeHTTP(writer, request)
	})
}

// doBearerJWTAuth parse and decrypt the token and verify the user.
func (m *Manager) doBearerJWTAuth(token string) error {
	tok, err := jwt.ParseSignedAndEncrypted(token)
	if err != nil {
		return err
	}

	nested, err := tok.Decrypt(m.config.EncryptKey)
	if err != nil {
		return err
	}

	var claims jwt.Claims
	if err := nested.Claims(m.config.SignKey, &claims); err != nil {
		return err
	}

	// check if claim expired
	if claims.Expiry.Time().Before(time.Now()) {
		return fmt.Errorf("cliam expired")
	}

	// check that claim is active
	if claims.NotBefore.Time().After(time.Now()) {
		return fmt.Errorf("claim used before active")
	}

	if _, err = m.store.GetUser(claims.Subject); err != nil {
		return fmt.Errorf("error getting subject '%s': %w", claims.Subject, err)
	}

	// otherwise assume the subject is correct
	return nil
}

// doBasicAuth will decode the user and password and verify them against the store.
func (m *Manager) doBasicAuth(encodedUserPass string) error {
	// badly encoded
	userPassString, err := base64.StdEncoding.DecodeString(encodedUserPass)
	if err != nil {
		return err
	}

	// to many parts
	parts := strings.SplitN(string(userPassString), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid user password string in basic auth")
	}

	// if we have basic auth then verify user/password. If this errors then the user does not exist
	userStruct, err := m.store.GetUser(parts[0])
	if err != nil {
		return err
	}

	// check password
	if !auth.CheckPassword(parts[1], userStruct.Password) {
		return fmt.Errorf("invalid credentials")
	}

	return nil
}

// loggingMiddleware logs all requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, PathUIRoot) {
			zap.S().Debug("(REST) ", r.Method, " ", r.URL.Path)
		} else {
			zap.S().Info("(REST) ", r.Method, " ", r.URL)
		}

		next.ServeHTTP(w, r)
	})
}

// initializedMiddleware sends 503 if the service is not initialized. The exception is the initialization endpoint which
// can be accessed even if not initialized.
func (m *Manager) initializedMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// not initialized and not UI or init endpoint
		if !m.initialized && request.URL.Path != "/api/v1/self" && !strings.HasPrefix(request.URL.Path, PathUIRoot) &&
			request.URL.Path != "/" {
			restutil.HandleErrorWithExtras(restutil.ErrorResponse{
				Status: http.StatusServiceUnavailable,
				Msg:    "manager not initialized",
			}, writer, nil)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func sendUnauthorized(w http.ResponseWriter, promptForBasicAuth bool) {
	if promptForBasicAuth {
		w.Header().Set("WWW-Authenticate", `Basic realm="Couchbase Multi Cluster Manager"`)
	}
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("401 Unauthorized\n"))
}
