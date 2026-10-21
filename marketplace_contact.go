package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type ContactRequest struct {
	SellerName   string `json:"seller_name"`
	ListingTitle string `json:"listing_title"`
	BuyerEmail   string `json:"buyer_email"`
	BuyerUpdate  string `json:"buyer_update"`
	OrderHandoff string `json:"order_handoff"`
	CaptchaToken string `json:"captcha_token"`
}

type CaptchaVerifier interface {
	VerifyCaptcha(context.Context, string, string) (bool, error)
}

type InboxSender interface {
	SendInboxEmail(context.Context, string, string, string, string) (string, error)
}

type ContactRouter struct {
	verifier CaptchaVerifier
	sender   InboxSender
	inbox    string
}

type RouteResult struct {
	MessageID string
	Rejected  bool
}

func (r ContactRouter) Route(ctx context.Context, form ContactRequest, ip string) (RouteResult, error) {
	if strings.TrimSpace(form.SellerName) == "" || strings.TrimSpace(form.ListingTitle) == "" || strings.TrimSpace(form.BuyerEmail) == "" || strings.TrimSpace(form.CaptchaToken) == "" {
		return RouteResult{}, fmt.Errorf("seller_name, listing_title, buyer_email, and captcha_token are required")
	}
	valid, err := r.verifier.VerifyCaptcha(ctx, form.CaptchaToken, ip)
	if err != nil {
		var businessError *infraiError
		if errors.As(err, &businessError) {
			return RouteResult{Rejected: true}, nil
		}
		return RouteResult{}, err
	}
	if !valid {
		return RouteResult{Rejected: true}, nil
	}
	text := fmt.Sprintf("Seller asset: %s\nListing: %s\nBuyer: %s\nBuyer update: %s\nOrder handoff: %s", form.SellerName, form.ListingTitle, form.BuyerEmail, form.BuyerUpdate, form.OrderHandoff)
	messageID, err := r.sender.SendInboxEmail(ctx, r.inbox, "Marketplace contact: "+form.ListingTitle, text, contactKey(form))
	if err != nil {
		return RouteResult{}, err
	}
	return RouteResult{MessageID: messageID}, nil
}

func contactKey(form ContactRequest) string {
	sum := sha256.Sum256([]byte(form.SellerName + "\n" + form.ListingTitle + "\n" + form.BuyerEmail + "\n" + form.OrderHandoff))
	return hex.EncodeToString(sum[:])
}
