---- MODULE PurchaseSettlement ----
(*
Several deliveries of one paid purchase reach ProcessPurchaseById at once:
a provider webhook and its redelivery, two overlapping CryptoPay cron
ticks, or a Telegram update replayed after a restart. Each delivery that
gets past the status check extends the subscription in Remnawave once.

Property: one purchase extends the subscription at most once.
Harm when broken: the payer gets the paid months twice.

Locked = FALSE is the flow before PurchaseRepository.Settle: a delivery
reads the status with FindById, and the row stays unlocked until MarkAsPaid.
Locked = TRUE is Settle: SELECT ... FOR UPDATE, then the Remnawave call
and the status update, then COMMIT, all in one transaction.
*)
EXTENDS Naturals

CONSTANTS Deliveries, Locked

VARIABLES status, extensions, pc, holder

vars == <<status, extensions, pc, holder>>

Free == "free"

TypeOK ==
    /\ status \in {"pending", "paid"}
    /\ extensions \in 0..10
    /\ pc \in [Deliveries -> {"read", "extend", "markPaid", "done"}]
    /\ holder \in Deliveries \cup {Free}

Init ==
    /\ status = "pending"
    /\ extensions = 0
    /\ pc = [d \in Deliveries |-> "read"]
    /\ holder = Free

CanEnter(d) == IF Locked THEN holder = Free ELSE TRUE
Enter(d) == IF Locked THEN holder' = d ELSE holder' = holder
Leave == IF Locked THEN holder' = Free ELSE holder' = holder

Read(d) ==
    /\ pc[d] = "read"
    /\ CanEnter(d)
    /\ IF status = "paid"
         THEN pc' = [pc EXCEPT ![d] = "done"] /\ Leave
         ELSE pc' = [pc EXCEPT ![d] = "extend"] /\ Enter(d)
    /\ UNCHANGED <<status, extensions>>

Extend(d) ==
    /\ pc[d] = "extend"
    /\ extensions' = extensions + 1
    /\ pc' = [pc EXCEPT ![d] = "markPaid"]
    /\ UNCHANGED <<status, holder>>

ExtendFails(d) ==
    /\ pc[d] = "extend"
    /\ pc' = [pc EXCEPT ![d] = "read"]
    /\ Leave
    /\ UNCHANGED <<status, extensions>>

MarkPaid(d) ==
    /\ pc[d] = "markPaid"
    /\ status' = "paid"
    /\ pc' = [pc EXCEPT ![d] = "done"]
    /\ Leave
    /\ UNCHANGED extensions

Finished ==
    /\ \A d \in Deliveries : pc[d] = "done"
    /\ UNCHANGED vars

Next ==
    \/ \E d \in Deliveries : Read(d) \/ Extend(d) \/ ExtendFails(d) \/ MarkPaid(d)
    \/ Finished

Spec == Init /\ [][Next]_vars

ExtendedAtMostOnce == extensions <= 1
====
