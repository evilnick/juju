// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package google

import (
	"encoding/json"
	"io"
	"net/mail"

	"github.com/juju/errors"
)

// The names of OS environment variables related to GCE.
//
// Note that these are not specified by Google. Instead they are
// defined by juju for use with the GCE provider. If Google defines
// equivalent environment variables they should be used instead.
const (
	OSEnvPrivateKey    = "GCE_PRIVATE_KEY"
	OSEnvClientID      = "GCE_CLIENT_ID"
	OSEnvClientEmail   = "GCE_CLIENT_EMAIL"
	OSEnvRegion        = "GCE_REGION"
	OSEnvProjectID     = "GCE_PROJECT_ID"
	OSEnvImageEndpoint = "GCE_IMAGE_URL"
)

// ParseAuthFile extracts the auth information from the JSON file
// downloaded from the GCE console (under /apiui/credential).
func ParseAuthFile(authFile io.Reader) (map[string]string, error) {
	data := make(map[string]string)
	if err := json.NewDecoder(authFile).Decode(&data); err != nil {
		return nil, errors.Trace(err)
	}
	for k, v := range data {
		switch k {
		case "private_key":
			data[OSEnvPrivateKey] = v
			delete(data, k)
		case "client_email":
			data[OSEnvClientEmail] = v
			delete(data, k)
		case "client_id":
			data[OSEnvClientID] = v
			delete(data, k)
		}
	}
	return data, nil
}

// Credentials holds the OAuth2 credentials needed to authenticate on GCE.
type Credentials struct {
	// ClientID is the GCE account's OAuth ID. It is part of the OAuth
	// config used in the OAuth-wrapping network transport.
	ClientID string

	// ClientEmail is the email address associatd with the GCE account.
	// It is used to generate a new OAuth token to use in the
	// OAuth-wrapping network transport.
	ClientEmail string

	// PrivateKey is the private key that matches the public key
	// associatd with the GCE account. It is used to generate a new
	// OAuth token to use in the OAuth-wrapping network transport.
	PrivateKey []byte
}

// NewCredentials returns a new Credentials based on the provided
// values. The keys must be recognized OS env var names for the
// different credential fields.
func NewCredentials(values map[string]string) (*Credentials, error) {
	var creds Credentials
	for k, v := range values {
		switch k {
		case OSEnvClientID:
			creds.ClientID = v
		case OSEnvClientEmail:
			creds.ClientEmail = v
		case OSEnvPrivateKey:
			creds.PrivateKey = []byte(v)
		default:
			return nil, errors.NotSupportedf("key %q", k)
		}
	}
	return &creds, nil
}

// Values returns the credentials as a simple mapping with the
// corresponding OS env variable names as the keys.
func (gc Credentials) Values() map[string]string {
	return map[string]string{
		OSEnvClientID:    gc.ClientID,
		OSEnvClientEmail: gc.ClientEmail,
		OSEnvPrivateKey:  string(gc.PrivateKey),
	}
}

// Validate checks the credentialss for invalid values. If the values
// are not valid, it returns errors.NotValid with the message set to
// the corresponding OS environment variable name.
//
// To be considered valid, each of the credentials must be set to some
// non-empty value. Furthermore, ClientEmail must be a proper email
// address.
func (gc Credentials) Validate() error {
	if gc.ClientID == "" {
		return NewMissingConfigValue(OSEnvClientID, "ClientID")
	}
	if gc.ClientEmail == "" {
		return NewMissingConfigValue(OSEnvClientEmail, "ClientEmail")
	}
	if _, err := mail.ParseAddress(gc.ClientEmail); err != nil {
		return NewInvalidConfigValue(OSEnvClientEmail, gc.ClientEmail, err)
	}
	if len(gc.PrivateKey) == 0 {
		return NewMissingConfigValue(OSEnvPrivateKey, "PrivateKey")
	}
	return nil
}

// ConnectionConfig contains the config values used for a connection
// to the GCE API.
type ConnectionConfig struct {
	// Region is the GCE region in which to operate for the connection.
	Region string

	// ProjectID is the project ID to use in all GCE API requests for
	// the connection.
	ProjectID string
}

// Validate checks the connection's fields for invalid values.
// If the values are not valid, it returns a config.InvalidConfigValue
// error with the key set to the corresponding OS environment variable
// name.
//
// To be considered valid, each of the connection's must be set to some
// non-empty value.
func (gc ConnectionConfig) Validate() error {
	if gc.Region == "" {
		return NewMissingConfigValue(OSEnvRegion, "Region")
	}
	if gc.ProjectID == "" {
		return NewMissingConfigValue(OSEnvProjectID, "ProjectID")
	}
	return nil
}
