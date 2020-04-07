package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseBearer(t *testing.T) {
	type args struct {
		auth string
	}
	tests := []struct {
		name      string
		args      args
		wantToken string
		wantSub   string
		wantErr   bool
	}{
		{
			name: "subNotFound",
			args: args{
				auth: "Bearer " + `eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IlFrWTRNekZHTVRkRk16TXlOME5HUWpFeU9VRkZNekU1UTBaRU1VWXpRVVF4TmtORU5UbEVNZyJ9.eyJpc3MiOiJodHRwczovL29jZmNsb3VkLmV1LmF1dGgwLmNvbS8iLCJzdWIiOiJRZkhoaE5LWEoxRjlTMXNRZHY3OTFNSm1FVndLc0J2cUBjbGllbnRzIiwiYXVkIjoiaHR0cHM6Ly9wb3J0YWwub2NmY2xvdWQuY29tLyIsImlhdCI6MTU1MDg0ODQ4NSwiZXhwIjoxNTUwOTM0ODg1LCJhenAiOiJRZkhoaE5LWEoxRjlTMXNRZHY3OTFNSm1FVndLc0J2cSIsImd0eSI6ImNsaWVudC1jcmVkZW50aWFscyJ9.Wv93hqC0branpOzm3_-T_DG2tQNYX8CvJntU7ZWK6J9BLzXqVx0Up-oH5lS_hShAVV3JjIoJ7CtACY_knWSkxvMjkeYd5Kcltm__XK9vK153RJyMHnc1EZFRR36ifH_Z6ewxYeUJoAcitNNJEQcaTkhavTBDe5-oXk-KT8Mtui0uzE18uO7Mdl0d2NN6mnd-2sJsC8LC5-rCOCPv3WRbEm76G_oBAllGhHf21bx4wP6iexZbhO1vofOSq4JfK_fdye4e86cmitCoQBUuOIV-Qrr8i_MiRVKVGdDxw1_wGsrjJcAr3NH4EiekGhQU4pbqHO5BJ8eJOXCs0OhGjXCQsw`,
			},
			wantErr:   false,
			wantToken: `eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IlFrWTRNekZHTVRkRk16TXlOME5HUWpFeU9VRkZNekU1UTBaRU1VWXpRVVF4TmtORU5UbEVNZyJ9.eyJpc3MiOiJodHRwczovL29jZmNsb3VkLmV1LmF1dGgwLmNvbS8iLCJzdWIiOiJRZkhoaE5LWEoxRjlTMXNRZHY3OTFNSm1FVndLc0J2cUBjbGllbnRzIiwiYXVkIjoiaHR0cHM6Ly9wb3J0YWwub2NmY2xvdWQuY29tLyIsImlhdCI6MTU1MDg0ODQ4NSwiZXhwIjoxNTUwOTM0ODg1LCJhenAiOiJRZkhoaE5LWEoxRjlTMXNRZHY3OTFNSm1FVndLc0J2cSIsImd0eSI6ImNsaWVudC1jcmVkZW50aWFscyJ9.Wv93hqC0branpOzm3_-T_DG2tQNYX8CvJntU7ZWK6J9BLzXqVx0Up-oH5lS_hShAVV3JjIoJ7CtACY_knWSkxvMjkeYd5Kcltm__XK9vK153RJyMHnc1EZFRR36ifH_Z6ewxYeUJoAcitNNJEQcaTkhavTBDe5-oXk-KT8Mtui0uzE18uO7Mdl0d2NN6mnd-2sJsC8LC5-rCOCPv3WRbEm76G_oBAllGhHf21bx4wP6iexZbhO1vofOSq4JfK_fdye4e86cmitCoQBUuOIV-Qrr8i_MiRVKVGdDxw1_wGsrjJcAr3NH4EiekGhQU4pbqHO5BJ8eJOXCs0OhGjXCQsw`,
			wantSub:   `QfHhhNKXJ1F9S1sQdv791MJmEVwKsBvq@clients`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, gotSub, err := parseBearer(tt.args.auth)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantToken, gotToken)
			assert.Equal(t, tt.wantSub, gotSub)
		})
	}
}
