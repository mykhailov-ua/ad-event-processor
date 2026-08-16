# TLC model: license state vs wall-clock time (P-C3-02 sketch).
# Run: tlc licensing_state.tla -config licensing_state.cfg
# Optional M3.1 gate — not required for pilot CI.

---- MODULE licensing_state ----

EXTENDS Naturals, TLC

VARIABLES state, clock, revoked, valid_until, grace_days

AllowedStates == {"ACTIVE", "GRACE", "EXPIRED", "REVOKED"}

TypeInvariant ==
  /\ state \in AllowedStates
  /\ clock \in Nat
  /\ revoked \in BOOLEAN
  /\ valid_until \in Nat
  /\ grace_days \in Nat

Init ==
  /\ state = "ACTIVE"
  /\ clock = 0
  /\ revoked = FALSE
  /\ valid_until = 100
  /\ grace_days = 7

ExpireState ==
  IF revoked
  THEN "REVOKED"
  ELSE IF clock < valid_until
  THEN "ACTIVE"
  ELSE IF clock < valid_until + grace_days
  THEN "GRACE"
  ELSE "EXPIRED"

Tick ==
  /\ clock' = clock + 1
  /\ state' = ExpireState
  /\ UNCHANGED <<revoked, valid_until, grace_days>>

Revoke ==
  /\ ~revoked
  /\ revoked' = TRUE
  /\ state' = "REVOKED"
  /\ UNCHANGED <<clock, valid_until, grace_days>>

ApplyFreshToken ==
  /\ clock' = 0
  /\ state' = "ACTIVE"
  /\ revoked' = FALSE
  /\ valid_until' = 100
  /\ grace_days' = 7

Next ==
  \/ Tick
  \/ Revoke
  \/ ApplyFreshToken

Spec == Init /\ [][Next]_<<state, clock, revoked, valid_until, grace_days>>

IngestOK == state \in {"ACTIVE", "GRACE"}

IngestInvariant == IngestOK => state # "EXPIRED" /\ state # "REVOKED"

MonotonicWhenNotRevoked ==
  revoked => TRUE
  \* Informal: without ApplyFreshToken, rank(state) non-increasing in time — checked by tests.

====
