#!/usr/bin/env bash
# CAP-F20: many-to-many relationship (Follow) -- neither side owns the row,
# both sides independently queryable, the (Follower, Followee) pair unique.
# Sourced by run.sh after lib.sh -- assumes lib.sh's helpers/ACCOUNTS are
# already in scope.
#
# Proven here as PURE COMPOSITION, no new mechanism: `user` fields (CAP-F05)
# + a composite `unique` Constraint (CAP-C12) + two CAP-V05/V09
# `$current_user`-filtered Views (one per direction) -- seeds/033_follow_lab.sql.

AMIR=$(session_for f20.amir@example.com password)
BUDI=$(session_for f20.budi@example.com password)
AMIR_ID=$(user_option_id "$BASE_URL/mch_f20_follow/new" "$AMIR" "Follow Lab Amir")
BUDI_ID=$(user_option_id "$BASE_URL/mch_f20_follow/new" "$AMIR" "Follow Lab Budi")

# T189 -- CAP-F20/CAP-F05: a Follow record with two independent `user`
# fields creates normally -- no reference-field/child-table mechanism
# involved, just two ordinary picker fields.
FOLLOW1_URL=$(post_redirect "$BASE_URL/mch_f20_follow" "fld_f20_follower=$AMIR_ID&fld_f20_followee=$BUDI_ID" "$AMIR")
[ -n "$FOLLOW1_URL" ]
check T189 "CAP-F20" "a Follow record (two independent user fields, neither owning the other) creates normally" $?

# T190 -- CAP-C12: the SAME (Follower, Followee) pair a second time is
# rejected -- CAP-F20's own forcing case for composite uniqueness, proven
# for real for the first time (CAP-C12's own T62 proof used a different
# Machine shape, Approval Step's (Document, Sequence)).
post_body_contains "$BASE_URL/mch_f20_follow" "fld_f20_follower=$AMIR_ID&fld_f20_followee=$BUDI_ID" "only once" "$AMIR"
check T190 "CAP-C12" "the exact same (Follower, Followee) pair is rejected as a duplicate" $?

# T191 -- CAP-F20 negative-of-the-negative: the REVERSED pair (Budi follows
# Amir back) is NOT blocked by T190's constraint -- proves the composite
# uniqueness check is direction-sensitive (order of the two fields matters),
# not accidentally treating an unordered pair as one entity.
FOLLOW2_URL=$(post_redirect "$BASE_URL/mch_f20_follow" "fld_f20_follower=$BUDI_ID&fld_f20_followee=$AMIR_ID" "$BUDI")
[ -n "$FOLLOW2_URL" ]
check T191 "CAP-F20" "the reversed pair (B follows A, after A follows B) is a different pair, not blocked" $?

# T192 -- CAP-V05/V09: "My Following" (filter: Follower = $current_user)
# shows who Amir follows (Budi, from T189) -- the same $current_user
# mechanism CAP-V05's own T74/T75 proof used, just applied to a
# non-owner_field, arbitrary `user` field on a join Machine.
body_contains "$BASE_URL/mch_f20_follow?view=vw_f20_following" "Follow Lab Budi" "$AMIR"
check T192 "CAP-V05" "the \"My Following\" filtered view shows who the acting user follows" $?

# T193 -- the OTHER direction of the same mechanism, on the SAME Machine:
# "My Followers" (filter: Followee = $current_user) shows who follows Amir
# (Budi, from T191's reverse follow) -- this is the real bidirectional-
# lookup path for a `user`-field join Machine; CAP-V06's own childLists
# only walks `reference`-typed fields, so it would NOT have found this.
body_contains "$BASE_URL/mch_f20_follow?view=vw_f20_followers" "Follow Lab Budi" "$AMIR"
check T193 "CAP-V05" "the \"My Followers\" filtered view (the OTHER field, same Machine) shows who follows the acting user" $?
