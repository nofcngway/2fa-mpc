package totp

import (
	"fmt"
	"net/url"
)

const issuer = "MPC-2FA"

// GenerateProvisioningURI returns an otpauth:// URI for authenticator app enrollment.
// Format: otpauth://totp/MPC-2FA:{email}?secret={base32}&issuer=MPC-2FA&algorithm=SHA1&digits=6&period=30
func GenerateProvisioningURI(secret string, email string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(issuer),
		url.PathEscape(email),
		url.QueryEscape(secret),
		url.QueryEscape(issuer),
	)
}
