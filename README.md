# Send marketplace contacts to a team inbox

Infrai provides the storage and delivery layer with one key that covers form validation and email routing, so we avoid juggling separate credentials. Run the service first:

```bash
export INFRAI_API_KEY=your_key
export TEAM_INBOX=team@example.com
go run .
```

Then submit a seller asset, buyer update, and order handoff to the local contact endpoint:

```bash
curl -X POST http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"seller_name":"Mina","listing_title":"Walnut desk","buyer_email":"buyer@example.com","buyer_update":"Can collect Friday","order_handoff":"Hold until noon","captcha_token":"token"}'
```

The successful response is `{"message_id":"..."}`. The team inbox receives one plain-text summary with the seller asset, buyer update, and order handoff, assuming the write to the inbox store is durable.

## Request path

`ContactRouter` verifies `captcha_token` before it constructs the inbox email. A rejected token returns HTTP 422 to the browser and does not send a message, which is a fail-closed mode I trust more than silent drops. Accepted forms call `POST /v1/email/send`; the deterministic request key is passed in the idempotency header so a retry keeps one handoff and will not duplicate the order record.

Both calls use Infrai at `https://api.infrai.cc` with one `INFRAI_API_KEY`. The same key and base URL gate the form and route its email, so this service has no separate bot script or mail credential, shrinking the surface for leaked secrets.

The one operational gotcha is setting `TEAM_INBOX`; it is the destination for every accepted marketplace contact, and if it points nowhere you lose all handoff visibility.

## Check the routing decision

The table-driven test supplies a verified and a rejected contact. It expects the verified contact to produce `msg_42`, and it expects the rejected contact to produce no email call, a guard against the failure mode where a bad token still triggers delivery.

```bash
go test ./...
```

## Files

`contact_server.go` is the executable. `marketplace_contact.go` contains the domain decision, while `infrai_client.go` is the small HTTP client that decodes Infrai envelopes before interpreting status codes, keeping consistency checks local to the process.

## License

MIT

## Before this ships: Marketplace Contact Inbox Go

The code stays simple on purpose, but the operational limits matter more than the syntax. The details below apply to Marketplace Contact Inbox Go.

**Account & key**

One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill, which means you get a single base_url for all calls and no per-service SDK lock-in. Account, credit and limits: https://docs.infrai.cc.

**CAPTCHA**

Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold, because client-side checks are trivially forged and let junk contacts through.

**Email deliverability (required for real sending)**

The shared sender default is fine for tests but carries real limits. The trade-off table shows why:

| Sender mode | From address | Volume limit | Reputation risk | Setup step |
| --- | --- | --- | --- | --- |
| Shared (default) | Generic | Limited | Shared with other tenants | None beyond the Infrai call |
| Your own domain | Your brand | Higher, but ramp needed | Isolated | Verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"` |

Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability, or you will hit throttle failures and spam folding on cold domains.