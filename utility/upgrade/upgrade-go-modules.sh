#!/bin/bash

#
# Helper script to upgrade golang dependencies
#

# set -e

usage() {
	local readonly EXIT_STATUS=$1

cat << HELP_USAGE
Usage: $(basename $0)

Description:
    Helper script to upgrade golang dependecies

Options:
    -h / --help     show help
    -a / --all      upgrade all dependecies (both direct and indirect)
HELP_USAGE

	exit ${EXIT_STATUS}
}

UPGRADE_ALL_DEPENDENCIES=0

while getopts "ah-:" optchar; do
	case "${optchar}" in
		-)
			case "${OPTARG}" in
				all)
					UPGRADE_ALL_DEPENDENCIES=1
					;;
				help)
					usage 0
					;;
				*)
					echo "ERROR: Unknown option --${OPTARG}" >&2
					echo ""
					usage 1
					;;
			esac
			;;
		a)
			UPGRADE_ALL_DEPENDENCIES=1
			;;
		h)
			usage 0
			;;
		*)
			echo "ERROR: Unknown option -${OPTARG}" >&2
			echo ""
			usage 1
			;;
	esac
done

readonly SCRIPT_PATH=$(cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P)
readonly CLOUD_PATH=$(realpath "${SCRIPT_PATH}/../..")

cd "${CLOUD_PATH}"

if [ ${UPGRADE_ALL_DEPENDENCIES} -eq 1 ]; then
	go get -u ./...
else
	readonly DEPENDENCIES=($(go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all))
	for DEP in "${DEPENDENCIES[@]}"; do
		if ! go get -d "$DEP"; then
			echo "ERROR: failed to upgrade ${DEP}" >&2
		fi
	done
fi

go mod tidy
