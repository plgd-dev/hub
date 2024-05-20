#!/usr/bin/env bash

set -e

SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
. ${SCRIPTPATH}/common.sh

load_config $@
HOST=${STANDBY_MEMBERS[0]}

PRIMARY_MEMBER=$(get_primary_member $HOST)
if [ "$PRIMARY_MEMBER" ]; then
    echo "Primary member is $PRIMARY_MEMBER"
    HOST=$PRIMARY_MEMBER
fi

wait_for_standby_members $HOST
FORCE="true"
if [ "$PRIMARY_MEMBER" ]; then
    FORCE="false"
fi
SECONDARY_MEMBERS=( $(get_secondary_members $HOST "${STANDBY_MEMBERS[@]}") )
echo "Setting secondary members ${SECONDARY_MEMBERS[@]}"
set_secondary_members "$HOST" "$FORCE" "${STANDBY_MEMBERS[@]}"
echo "Setting hidden members ${STANDBY_MEMBERS[@]}"
set_hidden_members "$HOST" "$FORCE" "${SECONDARY_MEMBERS[@]}"
NEW_PRIMARY_MEMBER=$(move_primary "$PRIMARY_MEMBER" "${SECONDARY_MEMBERS[@]}")
if [ "$NEW_PRIMARY_MEMBER" != "$PRIMARY_MEMBER" ]; then
    echo "Setting old primary member $PRIMARY_MEMBER to hidden via $NEW_PRIMARY_MEMBER"
    set_hidden_members "$NEW_PRIMARY_MEMBER" "false" $PRIMARY_MEMBER
fi

