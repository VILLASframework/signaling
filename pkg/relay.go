// SPDX-FileCopyrightText: 2023 Steffen Vogel <post@steffenvogel.de>
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec
	"encoding/base64"
	"fmt"
	"time"

	"github.com/VILLASframework/signaling/pkg/stun"
)

const (
	DefaultRelayTTL = 1 * time.Hour
)

type RelayInfo struct {
	URL   string
	Realm string

	Username string
	Password string

	TTL    time.Duration
	Secret string
}

func NewRelayInfo(arg string) (RelayInfo, error) {
	u, user, pass, q, err := stun.ParseURI(arg)
	if err != nil {
		return RelayInfo{}, fmt.Errorf("invalid URL: %w", err)
	}

	r := RelayInfo{
		URL:      u.String(),
		Secret:   q.Get("secret"),
		Username: user,
		Password: pass,
		TTL:      DefaultRelayTTL,
	}

	if t := q.Get("ttl"); t != "" {
		ttl, err := time.ParseDuration(t)
		if err != nil {
			return RelayInfo{}, fmt.Errorf("invalid TTL: %w", err)
		}

		r.TTL = ttl
	}

	return r, nil
}

func NewRelayInfos(args []string) ([]RelayInfo, error) {
	relays := []RelayInfo{}
	for _, arg := range args {
		relay, err := NewRelayInfo(arg)
		if err != nil {
			return nil, err
		}

		relays = append(relays, relay)
	}

	return relays, nil
}

func (s *RelayInfo) GetCredentials(username string) (string, string, time.Time) {
	if s.Username != "" && s.Password != "" {
		return s.Username, s.Password, time.Time{}
	} else if s.Secret != "" {
		if s.Username != "" {
			username = s.Username
		}

		exp := time.Now().Add(s.TTL)
		user := fmt.Sprintf("%d:%s", exp.Unix(), username)

		digest := hmac.New(sha1.New, []byte(s.Secret))
		digest.Write([]byte(user))

		passRaw := digest.Sum(nil)
		pass := base64.StdEncoding.EncodeToString(passRaw)

		return user, pass, exp
	}

	return "", "", time.Time{}
}
