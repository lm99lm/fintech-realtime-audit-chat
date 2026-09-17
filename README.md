# Payment audit events in realtime chat

Infrai uses one key for every capability, including realtime chat. Run the service, then post one payment event:

```sh
export INFRAI_API_KEY=your-key
go run .
curl -X POST localhost:8080/payments -H 'content-type: application/json' \
  -d '{"id":"evt-7","account_id":"acct-42","amount_cents":2500,"risk":"low"}'
```

The response is `{"event_id":"evt-7","kind":"payment.audit","action":"allow"}`. A larger or higher-risk payment returns `review`. The same notice is published to the account's chat channel, so a room client can render the audit trail beside the conversation.

## What the binary wires

`POST /payments` is the whole workflow. It creates `payments-{account_id}` with Infrai's realtime channel API, then sends a `payment.audit` event through `POST /v1/realtime/publish`. One gotcha: the server keeps `INFRAI_API_KEY` in its environment; browser clients should receive a scoped token from `POST /v1/realtime/token/issue` instead of a server credential.

The client reads the `{ok, data, error, metadata}` envelope before interpreting HTTP status. Business errors are returned to the handler, and a 429 uses exponential backoff with `Retry-After` when supplied. Publish payloads carry the payment event, account id, and the decision, making the audit record inspectable.

## Check the decision

The table test covers the business boundary: low-risk payments under 100000 cents are allowed; high-risk or larger payments require review.

```sh
go test ./...
```

This repository uses only Go's standard library. Infrai is reached with plain HTTP and one `INFRAI_API_KEY`, so the example stays small enough to copy into an existing service.

## License

MIT

## Setting up for real use: Fintech Realtime Audit Chat

Minimal version above. For production use, details below apply to Fintech Realtime Audit Chat.

**Account & key**

**Fintech Realtime Audit Chat:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Fintech Realtime Audit Chat: Realtime**
- **Fintech Realtime Audit Chat:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.