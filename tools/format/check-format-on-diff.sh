#!/bin/bash

#
# Check whether changed files in current branch are correctly formatted
#

set -e

usage() {
    local readonly EXIT_STATUS=$1

cat << HELP_USAGE
Usage: $(basename $0) [-b|--branch BRANCH_NAME]

Description:
    Helper script to check formatting of changed files in current branch

Options:
    -b / --branch       branch name to be used as base for comparison (master if not set)
    -h / --help         show help
    -s / --simplify     tell gofmt to also simplify code
HELP_USAGE

    exit ${EXIT_STATUS}
}

BASE_BRANCH=master
SIMPLIFY=0

while getopts "b:hs-:" optchar; do
    case "${optchar}" in
        -)
            case "${OPTARG}" in
                branch)
                    BASE_BRANCH="${!OPTIND}"; OPTIND=$(($OPTIND+1))
                    ;;
                help)
                    usage 0
                    ;;
                simplify)
                    SIMPLIFY=1
                    ;;
                *)
                    echo "ERROR: Unknown option --${OPTARG}" >&2
                    echo ""
                    usage 1
                    ;;
            esac
            ;;
        b)
            BASE_BRANCH=${OPTARG}
            ;;
        h)
            usage 0
            ;;
        s)
            SIMPLIFY=1
            ;;
        *)
            echo "ERROR: Unknown option -${OPTARG}" >&2
            echo ""
            usage 1
            ;;
    esac
done

shift $((OPTIND-1))

if [[ -z "${BASE_BRANCH}" ]]; then
    echo "ERROR: empty base branch" >&2
    exit 1
fi

readonly SCRIPT_DIR=$(cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P)
readonly CLOUD_DIR=$(realpath "${SCRIPT_DIR}/../..")

cd "${CLOUD_DIR}"

if ! BASE_COMMIT=$(/usr/bin/git merge-base HEAD ${BASE_BRANCH}); then
    echo "ERROR: failed to obtain base commit" >&2
    exit 1
fi

if ! CHANGED_FILES_OUTPUT=$(/usr/bin/git diff --diff-filter=ACMR --name-only HEAD ${BASE_COMMIT}); then
    echo "ERROR: failed to obtain list of changed files" >&2
    exit 1
fi
CHANGED_FILES=(${CHANGED_FILES_OUTPUT})

if [[ ${#CHANGED_FILES[@]} -eq 0 ]]; then
    exit 0
fi

for FILE in "${CHANGED_FILES[@]}"; do
    if [[ "${FILE}" =~ .go$ ]]; then
        OPTS=(-w)
        if [[ ${SIMPLIFY} -eq 1 ]]; then
            OPTS+=(-s)
        fi
        gofmt ${OPTS[@]} "${FILE}"
    fi
done

UNFORMATTED_FILES=($(git diff --name-only))

if [[ ${#UNFORMATTED_FILES[@]} -eq 0 ]]; then
    echo "All go files are formatted correctly"
    exit 0
fi

echo "Unformatted files detected:"
for FILE in "${UNFORMATTED_FILES[@]}"; do
    echo "  ${FILE}"
done
