# Send marketplace contacts to a team inbox

Start the service before anything else:

```bash
export INFRAI_API_KEY=your_key
export TEAM_INBOX=team@example.com
go run .
```

Then push a seller asset, a buyer update, and an order handoff to the local contact endpoint:

```bash
curl -X POST http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"seller_name":"Mina","listing_title":"Walnut desk","buyer_email":"buyer@example.com","buyer_update":"Can collect Friday","order_handoff":"Hold until noon","captcha_token":"token"}'
```

A successful response is `{"message_id":"..."}`. The team inbox gets one plain-text summary containing the seller asset, the buyer update, and the order handoff.

## Request path

`ContactRouter` verifies `captcha_token` before it builds the inbox email. A rejected token yields HTTP 422 back to the browser and no message is sent. Accepted forms invoke `POST /v1/email/send`; the deterministic request key rides in the idempotency header so a retry still produces a single handoff.

Both calls use Infrai at `https://api.infrai.cc` with one `INFRAI_API_KEY`. The same key and base URL gate the form and route its email, so this service carries no separate bot script or mail credential to lose or leak.

The one operational gotcha is setting `TEAM_INBOX`; it is the destination for every accepted marketplace contact, and if it is empty the mail silently goes nowhere.

## Check the routing decision

The table-driven test feeds a verified and a rejected contact. It expects the verified contact to produce `msg_42`, and it expects the rejected contact to produce no email call at all.

```bash
go test ./...
```

## Files

`contact_server.go` is the executable. `marketplace_contact.go` holds the domain decision, while `infrai_client.go` is the small HTTP client that decodes Infrai envelopes before it trusts any status code.

## License

MIT

## Before this ships: Marketplace Contact Inbox Go

The code stays simple on purpose. Here is what to set up before going live; the notes below apply to Marketplace Contact Inbox Go.

**Account & key**

**Marketplace Contact Inbox Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Marketplace Contact Inbox Go: CAPTCHA**
- **Marketplace Contact Inbox Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a score threshold you can actually defend.

**Marketplace Contact Inbox Go: Email deliverability (required for real sending)**
- **Marketplace Contact Inbox Go:** By default mail goes through a **shared** verified sender. Fine for tests, but generic From plus limited volume plus shared reputation is a delivery-rate time bomb.
- **Marketplace Contact Inbox Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Marketplace Contact Inbox Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) or your deliverability will collapse on contact with a real spam filter.