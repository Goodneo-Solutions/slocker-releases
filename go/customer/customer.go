// Package customer is the customer side of sLocker: the smart-locker operations of the partner app API
// (open.superroute.ca/app/v1) — pickups, locker locations, locker storage, the locker shop and locker resale — on top
// of the sApp core (github.com/Goodneo-Solutions/sapp-releases/go). It is its own module, so the carrier module
// (github.com/Goodneo-Solutions/slocker-releases/go) never depends on sApp.
//
//	locker, err := customer.New(customer.Options{AppCode: "p_acme_parcels", ClientCode: "k3m9q2", AppKey: "sp_appkey_…"})
//	_, err = locker.SApp.SignInWithIDToken(ctx, googleIDToken, nil)
//	pickups, err := locker.API.PlatformAppPickups(ctx, customer.PlatformAppPickupsParams{Status: sapp.String("ready")})
//
//	locker := customer.Share(sr) // the sign-in of an sApp client the app already has
package customer

import (
	"context"
	"errors"

	sapp "github.com/Goodneo-Solutions/sapp-releases/go"
)

// operationCall is what LockerAPI calls for each operation: sApp's PartnerClient.CallOperation.
type operationCall func(ctx context.Context, op sapp.Operation, params map[string]any, body map[string]any, files []sapp.File, opts ...sapp.CallOption) (any, error)

// Options are the options of the sApp partner client New makes (app code, client code, app key, token store, …).
type Options = sapp.PartnerOptions

// Client is the customer side of sLocker over one sApp partner client.
type Client struct {
	// API has one method per smart-locker operation, named as in sApp 0.2.1 (PlatformAppPickups,
	// PlatformAppCollectPickup, …), and the locker resale operations.
	API *LockerAPI
	// SApp is the partner client underneath: sign-in and the session, shared with every other partner app call.
	SApp *sapp.PartnerClient
}

// New makes the sApp partner client from o and answers the customer side over it.
func New(o Options) (*Client, error) {
	sr, err := sapp.NewPartnerClient(o)
	if err != nil {
		return nil, err
	}
	return Share(sr), nil
}

// Share answers the customer side over a partner client the app already has, so one sign-in serves both.
func Share(sr *sapp.PartnerClient) *Client {
	if sr == nil {
		panic(errors.New("customer.Share: the sApp client is nil"))
	}
	return &Client{API: &LockerAPI{call: sr.CallOperation}, SApp: sr}
}

// CallOperation calls an operation described by hand through the sApp pipeline (see sapp.PartnerClient.CallOperation).
func (c *Client) CallOperation(ctx context.Context, op sapp.Operation, params map[string]any, body map[string]any, files []sapp.File, opts ...sapp.CallOption) (any, error) {
	return c.SApp.CallOperation(ctx, op, params, body, files, opts...)
}
