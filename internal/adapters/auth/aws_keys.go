package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// AWSKeysProvider prompts for AWS access key + secret + optional session token,
// then validates them via STS GetCallerIdentity.
//
// The credential is stored as:
//
//	AccessToken  = access_key_id
//	RefreshToken = json{"secret":"...", "session_token":"..."}
type AWSKeysProvider struct {
	transport ports.Transport
}

// NewAWSKeysProvider creates an AWS keys provider. Use WithValidation to enable STS verification.
func NewAWSKeysProvider() *AWSKeysProvider { return &AWSKeysProvider{} }

// WithValidation supplies the transport used to call STS GetCallerIdentity at login.
func (p *AWSKeysProvider) WithValidation(transport ports.Transport) *AWSKeysProvider {
	p.transport = transport
	return p
}

func (p *AWSKeysProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderAWSKeys
}

type awsKeyPayload struct {
	Secret       string `json:"secret"`
	SessionToken string `json:"session_token,omitempty"`
}

func (p *AWSKeysProvider) Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	akid, err := promptSecret("AWS access key ID (input hidden): ")
	if err != nil {
		return core.Credential{}, err
	}
	secret, err := promptSecret("AWS secret access key (input hidden): ")
	if err != nil {
		return core.Credential{}, err
	}
	// Session token is optional; empty input is OK here.
	session, sessionErr := promptSecret("AWS session token (optional; press enter to skip): ")
	payload := awsKeyPayload{Secret: secret.Reveal()}
	if sessionErr == nil {
		payload.SessionToken = session.Reveal()
	}
	raw, _ := json.Marshal(payload)
	cred := core.Credential{
		Provider:     core.ProviderAWSKeys,
		Service:      svc,
		Account:      alias,
		AccessToken:  akid,
		RefreshToken: core.NewSecret(string(raw)),
	}
	if p.transport != nil {
		if err := stsGetCallerIdentity(ctx, p.transport, akid.Reveal(), payload.Secret, payload.SessionToken); err != nil {
			return core.Credential{}, fmt.Errorf("aws_keys: STS validation failed: %w", err)
		}
	}
	return cred, nil
}

func (p *AWSKeysProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil
}

// stsGetCallerIdentity signs and sends a GetCallerIdentity request to verify the keys.
// Implements just enough of AWS Signature Version 4 to call STS.
func stsGetCallerIdentity(ctx context.Context, tr ports.Transport, akid, secret, session string) error {
	const (
		host    = "sts.amazonaws.com"
		region  = "us-east-1"
		service = "sts"
	)
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	query := "Action=GetCallerIdentity&Version=2011-06-15"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", host, amzDate)
	signedHeaders := "host;x-amz-date"
	if session != "" {
		canonicalHeaders = fmt.Sprintf("host:%s\nx-amz-date:%s\nx-amz-security-token:%s\n", host, amzDate, session)
		signedHeaders = "host;x-amz-date;x-amz-security-token"
	}
	payloadHash := sha256Hex("")
	canonicalRequest := strings.Join([]string{
		"GET", "/", query, canonicalHeaders, signedHeaders, payloadHash,
	}, "\n")

	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := strings.Join([]string{
		algorithm, amzDate, credentialScope, sha256Hex(canonicalRequest),
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+secret), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	auth := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, akid, credentialScope, signedHeaders, signature)

	url := "https://" + host + "/?" + query
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Authorization", auth)
	if session != "" {
		req.Header.Set("X-Amz-Security-Token", session)
	}
	resp, err := tr.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("STS returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

var _ ports.AuthProvider = (*AWSKeysProvider)(nil)
