#!/usr/bin/env bash

set -e

STANDBY_MEMBERS=()
STANDBY_DELAY_SECS=10
MODE="standby"

SECONDARY_VOTES=1
SECONDARY_PRIORITY=10
SECONDARY_DELAY_SECS=0

TLS_ENABLED=false
TLS_CA=""
TLS_USE_SYSTEM_CA=false
TLS_CERTIFICATE_KEY_FILE="/tmp/tlsCertificateKeyFile.pem"

load_config() {
    local CONFIG_FILE=""
    while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
        CONFIG_FILE="$2"
        shift # past argument
        shift # past value
        ;;
        -*|--*)
        echo "Unknown option $1"
        exit 1
        ;;
        *)
    esac
    done
    echo "Loading config from \"$CONFIG_FILE\""
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "Config file \"$CONFIG_FILE\" not found"
        exit 1
    fi

    echo "Loading MODE"
    MODE=$(yq -e eval '.mode' "$CONFIG_FILE")
    if [ "$MODE" != "standby" ] && [ "$MODE" != "active" ]; then
        echo "Invalid .mode value \"$MODE\", mode must be either 'standby' or 'active'"
        exit 1
    fi

    echo "Loading STANDBY_MEMBERS"
    STANDBY_MEMBERS=($(yq -e -r '.replicaSet.standby.members[]' "$CONFIG_FILE"))
    if [ ${#STANDBY_MEMBERS[@]} -eq 0 ]; then
        echo "No standby members found in config file"
        exit 1
    fi
    
    echo "Loading STANDBY_DELAY_SECS"
    STANDBY_DELAY_SECS=$(yq -e eval '.replicaSet.standby.delaysSeconds' "$CONFIG_FILE")
    if [ -z "$STANDBY_DELAY_SECS" ]; then
        STANDBY_DELAY_SECS=10
    fi
    if [ $STANDBY_DELAY_SECS -lt 0 ]; then
        echo "Invalid .replicaSet.standby.delaysSeconds value"
        exit 1
    fi

    echo "Loading SECONDARY_DELAY_SECS"
    SECONDARY_DELAY_SECS=$(yq -e eval '.replicaSet.secondary.delaysSeconds' "$CONFIG_FILE")
    if [ -z "$SECONDARY_DELAY_SECS" ]; then
        SECONDARY_DELAY_SECS=0
    fi
    if [ $SECONDARY_DELAY_SECS -lt 0 ]; then
        echo "Invalid .replicaSet.secondary.delaysSeconds value"
        exit 1
    fi

    echo "Loading SECONDARY_VOTES"
    SECONDARY_VOTES=$(yq -e eval '.replicaSet.secondary.votes' "$CONFIG_FILE")
    if [ -z "$SECONDARY_VOTES" ]; then
        SECONDARY_VOTES=1
    fi
    if [ $SECONDARY_VOTES -lt 1 ]; then
        echo "Invalid .replicaSet.secondary.votes value"
        exit 1
    fi

    echo "Loading SECONDARY_PRIORITY"
    SECONDARY_PRIORITY=$(yq -e eval '.replicaSet.secondary.priority' "$CONFIG_FILE")
    if [ -z "$SECONDARY_PRIORITY" ]; then
        SECONDARY_PRIORITY=10
    fi
    if [ $SECONDARY_PRIORITY -lt 1 ]; then
        echo "Invalid .replicaSet.secondary.priority value"
        exit 1
    fi

    echo "Loading TLS"
    TLS_ENABLED=$(yq eval '.clients.storage.mongoDB.tls.enabled' "$CONFIG_FILE")
    if [ "$TLS_ENABLED" == "true" ]; then
        echo "Loading TLS_CA"
        TLS_CA=$(yq -e eval '.clients.storage.mongoDB.tls.caPool' "$CONFIG_FILE")
        echo "Loading TLS_CERT"
        local TLS_CERT=$(yq -e eval '.clients.storage.mongoDB.tls.certFile' "$CONFIG_FILE")
        echo "Loading TLS_KEY"
        local TLS_KEY=$(yq -e eval '.clients.storage.mongoDB.tls.keyFile' "$CONFIG_FILE")
        if [ -z $TLS_CERT ] && [ -z $TLS_KEY ]; then
            rm -f $TLS_CERTIFICATE_KEY_FILE
        else
            cat $TLS_CERT $TLS_KEY > $TLS_CERTIFICATE_KEY_FILE
        fi
        echo "Loading TLS_USE_SYSTEM_CA"
        TLS_USE_SYSTEM_CA=$(yq eval '.clients.storage.mongoDB.tls.useSystemCAPool' "$CONFIG_FILE")
    fi
    echo "Loaded config"
    echo "  HOST: $HOST"
    echo "  STANDBY_MEMBERS: ${STANDBY_MEMBERS[@]}"
    echo "  STANDBY_DELAY_SECS: $STANDBY_DELAY_SECS"
    echo "  SECONDARY_DELAY_SECS: $SECONDARY_DELAY_SECS"
    echo "  SECONDARY_VOTES: $SECONDARY_VOTES"
    echo "  SECONDARY_PRIORITY: $SECONDARY_PRIORITY"
    echo "  TLS_ENABLED: $TLS_ENABLED"
    echo "  TLS_CA: $TLS_CA"
    echo "  TLS_USE_SYSTEM_CA: $TLS_USE_SYSTEM_CA"
    echo "  TLS_CERTIFICATE_KEY_FILE: $TLS_CERTIFICATE_KEY_FILE"

    return 0
}

get_tls_args() {
    local TLS_ARGS=""
    if [ "$TLS_ENABLED" == "true" ]; then
        TLS_ARGS="--tls"
        if [ -f "$TLS_CA" ]; then
            TLS_ARGS="$TLS_ARGS --tlsCAFile $TLS_CA"
        fi
        if [ -f "$TLS_CERTIFICATE_KEY_FILE" ]; then
            TLS_ARGS="$TLS_ARGS --tlsCertificateKeyFile $TLS_CERTIFICATE_KEY_FILE"
        fi
        if [ "$TLS_USE_SYSTEM_CA" == "true" ]; then
            TLS_ARGS="$TLS_ARGS --tlsUseSystemCA"
        fi
    fi
    echo $TLS_ARGS
}


get_status() {
  echo $(mongosh --host $1 $(get_tls_args) --eval "EJSON.stringify(rs.status())")
}

get_config() {
  echo $(mongosh --host $1 $(get_tls_args) --eval "EJSON.stringify(rs.conf())")
}

get_primary_member_from_status() {
    local STATUS=$1
    echo $(echo $STATUS | jq -r '.members[] | select(.state == 1) | .name')
}

get_primary_member() {
    local HOST=$1
    local STATUS=$(get_status $HOST)
    get_primary_member_from_status "$STATUS"
}

wait_for_standby_members() {
    local HOST=$1
    while true; do
        echo "Checking if all backup members are ready"
        local MEMBERS_EXISTS=true
        local CONFIG=$(get_config $HOST)
        echo $CONFIG
        for MEMBER in "${STANDBY_MEMBERS[@]}"; do
            echo 'Checking member "'$MEMBER'"'
            if ! echo $CONFIG | jq -e -r '.members[] | select(.host == "'$MEMBER'")' > /dev/null; then
                echo "Member $MEMBER is not ready"
                MEMBERS_EXISTS=false
            else
                echo "Member $MEMBER is ready"
            fi
        done
        if [ "$MEMBERS_EXISTS" == "true" ]; then
            break
        fi
        sleep 1
    done
}

get_secondary_members() {
    local HOST=$1
    shift
    local STANDBY_MEMBERS=("$@")
    local CONFIG=$(get_config $HOST)
    local MEMBERS_SERVERS=($(echo $(echo $CONFIG | jq -r '.members[] | .host')))
    local SECONDARY_SERVERS=()
    for MEMBER in "${MEMBERS_SERVERS[@]}"; do 
        local IS_BACKUP_MEMBER=false
        for BACKUP_MEMBER in "${STANDBY_MEMBERS[@]}"; do
            if [ "$MEMBER" == "$BACKUP_MEMBER" ]; then
                IS_BACKUP_MEMBER=true
                continue
            fi
        done
        if [ "$IS_BACKUP_MEMBER" == "true" ]; then
            continue
        fi
        SECONDARY_SERVERS+=($MEMBER)
    done
    echo ${SECONDARY_SERVERS[*]}
}

set_secondary_members() {
    local HOST=$1
    shift
    local FORCE=$1
    shift
    local SECONDARY_SERVERS=("$@")
    for MEMBER in "${SECONDARY_SERVERS[@]}"; do 
        local CONFIG=$(get_config $HOST)
        if echo $CONFIG | jq -e -r '.members[] | select(.host == "'$MEMBER'" and .hidden == false and .priority > 0 and .votes > 0 and .secondaryDelaySecs == '$SECONDARY_DELAY_SECS')' > /dev/null; then
            echo "Member $MEMBER is correctly configured"
        else
            echo "Member $MEMBER is not correctly configured"
            echo "Configuring member $MEMBER"
            local NEW_CONFIG="$(echo $CONFIG | jq -e -r '(.members |= map(if .host == "'$MEMBER'" then .hidden = false | .priority = '$SECONDARY_PRIORITY' | .votes = '$SECONDARY_VOTES' | .secondaryDelaySecs = '$SECONDARY_DELAY_SECS' else . end))')"
            local OUT=$(mongosh --host $HOST $(get_tls_args) --eval "rs.reconfig(EJSON.deserialize($NEW_CONFIG),{force:$FORCE})")
            if [ $? -ne 0 ]; then
                echo "Failed to step down primary member $HOST"
                echo $OUT
                exit 1
            fi
        fi
    done
}

set_hidden_members() {
    local HOST=$1
    shift
    local FORCE=$1
    shift
    local HIDDEN_MEMBERS=("$@")
    echo "Setting backup members"
    for MEMBER in "${HIDDEN_MEMBERS[@]}"; do
        CONFIG=$(get_config $HOST)
        if echo $CONFIG | jq -e -r '.members[] | select(.host == "'$MEMBER'" and .hidden == true and .priority == 0 and .secondaryDelaySecs == '$STANDBY_DELAY_SECS' and .votes == 0)' > /dev/null; then
            echo "Member $MEMBER is correctly configured"
        elif [ "$MEMBER" == "$HOST" ] && [ "$FORCE" == "false" ]; then
            echo "Member $MEMBER is primary member, setting priority to 0.1"
            NEW_CONFIG="$(echo $CONFIG | jq -e -r '(.members |= map(if .host == "'$MEMBER'" then .priority = 0.1 else . end))')"
            local OUT=$(mongosh --host $HOST $(get_tls_args) --eval "rs.reconfig(EJSON.deserialize($NEW_CONFIG), {force:false})")
            if [ $? -ne 0 ]; then
                echo "Failed to step down primary member $HOST"
                echo $OUT
                exit 1
            fi
            continue
        else
            echo "Member $MEMBER is not correctly configured"
            echo "Configuring member $MEMBER"
            NEW_CONFIG="$(echo $CONFIG | jq -e -r '(.members |= map(if .host == "'$MEMBER'" then .hidden = true | .priority = 0 | .votes = 0 | .secondaryDelaySecs = '$STANDBY_DELAY_SECS' else . end))')"
            local OUT=$(mongosh --host $HOST $(get_tls_args) --eval "rs.reconfig(EJSON.deserialize($NEW_CONFIG), {force:$FORCE})")
            if [ $? -ne 0 ]; then
                echo "Failed to step down primary member $HOST"
                echo $OUT
                exit 1
            fi
        fi
    done
}

move_primary() {
    local HOST=$1
    shift
    local STANDBY_MEMBERS=("$@")
    local NUM_NOT_PRIMARY=0
    while true; do
        STATUS=$(get_status $HOST)
        HOST=$(echo $STATUS | jq -r '.members[] | select(.state == 1) | .name')
        if [ -z "$HOST" ]; then
            NUM_NOT_PRIMARY=$((NUM_NOT_PRIMARY+1))
            if [ "$NUM_NOT_PRIMARY" -gt 10 ]; then
                echo "Primary member not found after 10 attempts"
                exit 1
            fi
            sleep 1
            continue
        fi
        NUM_NOT_PRIMARY=0
        MOVED=true
        for MEMBER in "${STANDBY_MEMBERS[@]}"; do
            if [ "$HOST" == "$MEMBER" ]; then
                MOVED=false
                break
            fi
        done
        if [ "$MOVED" == "true" ]; then
           echo $HOST
           break
        fi
        local OUT=$(mongosh --host $HOST $(get_tls_args) --eval 'rs.stepDown()')
        if [ $? -ne 0 ]; then
            echo "Failed to step down primary member $HOST"
            echo $OUT
            exit 1
        fi
        sleep 1
    done
}