package main

import (
	"context"
	"testing"
)

type fakeVerifier struct{ valid bool }

func (f fakeVerifier) VerifyCaptcha(context.Context, string, string) (bool, error) {
	return f.valid, nil
}

type fakeSender struct{ calls int }

func (f *fakeSender) SendInboxEmail(_ context.Context, _, _, _, _ string) (string, error) {
	f.calls++
	return "msg_42", nil
}

func TestContactRouterRoutesOnlyVerifiedContact(t *testing.T) {
	for _, tc := range []struct {
		name      string
		verified  bool
		wantID    string
		wantCalls int
	}{
		{name: "verified contact reaches team inbox", verified: true, wantID: "msg_42", wantCalls: 1},
		{name: "rejected captcha never sends inbox email", verified: false, wantCalls: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender := &fakeSender{}
			router := ContactRouter{verifier: fakeVerifier{valid: tc.verified}, sender: sender, inbox: "team@example.com"}
			got, err := router.Route(context.Background(), ContactRequest{SellerName: "Mina", ListingTitle: "Walnut desk", BuyerEmail: "buyer@example.com", BuyerUpdate: "Can collect Friday", OrderHandoff: "Hold until noon", CaptchaToken: "token"}, "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			if got.MessageID != tc.wantID || sender.calls != tc.wantCalls || got.Rejected == tc.verified {
				t.Fatalf("result=%+v calls=%d", got, sender.calls)
			}
		})
	}
}
