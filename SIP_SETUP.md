# SIP Setup — Free Missed Call Delivery

The missed call channel dials the recipient via SIP, waits for a ring (180/183),
then immediately cancels. The recipient sees a missed call from your SIP number,
Googles it, and lands on your OpenCall page.

---

## Option 1 — Linphone (Easiest, completely free)

1. Go to https://www.linphone.org/freesip
2. Create a free account — no credit card required
3. You get: `yourusername@sip.linphone.org`
4. Set these env vars on Render (or wherever you deploy):

```
SIP_SERVER=sip.linphone.org
SIP_USERNAME=yourusername
SIP_PASSWORD=yourpassword
SIP_FROM=yourusername@sip.linphone.org
```

**Limitation:** Linphone SIP accounts do not come with a PSTN number —
the recipient will see a SIP URI, not a regular phone number. Fine for
VoIP-to-VoIP calls; for calls to real mobile numbers use Option 3.

---

## Option 2 — Google Voice (US numbers, free)

1. Go to https://voice.google.com
2. Claim a free US number — no credit card
3. In Google Voice settings, enable SIP/forwarding
4. Link the number to Linphone or another SIP app to obtain SIP credentials
5. Use the same env vars as Option 1, pointing at the SIP proxy Google Voice provides

**Limitation:** Google Voice SIP access is unofficial and may break without
notice. Good for personal testing; not recommended for production.

---

## Option 3 — VoIP.ms (cheapest paid — $0.001/min)

1. Sign up at https://voip.ms — deposit $1 minimum (no monthly fee)
2. Purchase a DID (real phone number) — typically $0.85/month
3. Recipients see a real caller ID they can Google
4. Set env vars using your VoIP.ms sub-account credentials:

```
SIP_SERVER=sip.voip.ms
SIP_USERNAME=yoursubaccount
SIP_PASSWORD=yourpassword
SIP_FROM=+1XXXXXXXXXX
```

**Recommended for production** — real PSTN number, reliable infrastructure,
essentially free at missed-call volumes (call is cancelled after 1-3 s).

---

## Missed Call Flow

```
Your server dials recipient
          |
          v
Their phone rings once (1-3 seconds)
          |
          v
Server sends CANCEL immediately
          |
          v
Recipient sees: "Missed call from +1XXXXXXXXXX"
          |
          v
They Google that number
          |
          v
Lands on opencall.net/missed?from=+1XXXXXXXXXX
          |
          v
Page says: "Someone on OpenCall tried to reach you"
          |
          v
Big button: "Answer their call"
```

---

## Testing the endpoint

Once env vars are set, trigger a missed call with:

```
GET /reach/missed-call?to=+1XXXXXXXXXX&inviteURL=https://opencall-server.vercel.app/join/abc123
```

Expected success response:
```json
{
  "success": true,
  "channel": "missed_call",
  "message": "Missed call sent — they will see your number"
}
```

Expected response when SIP is not yet configured:
```json
{
  "success": false,
  "error": "SIP_USERNAME not set",
  "fallback": "sip_not_configured"
}
```

---

## Env var reference

| Variable | Required | Default | Description |
|---|---|---|---|
| `SIP_SERVER` | No | `sip.linphone.org` | SIP registrar hostname |
| `SIP_USERNAME` | Yes | — | SIP account username |
| `SIP_PASSWORD` | Yes | — | SIP account password |
| `SIP_FROM` | Yes | — | Caller ID shown to recipient |
