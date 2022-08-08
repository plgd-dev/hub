#!/bin/bash

#
# Helper script to uprade repository dependencies:
#		- golang modules
#		- git submodules
#

set -e

usage() {
	local readonly EXIT_STATUS=$1

cat << HELP_USAGE
Usage: $(basename $0) [-aghs]

Description:
    Helper script to upgrade repository dependecies

Options:
    -h / --help             show help
    -g / --golang-only      upgrade only golang dependencies
    -a / --golang-all       upgrade all golang dependecies (both direct and indirect)
    -s / --submodules-only  upgrade only git submodules
HELP_USAGE

	exit ${EXIT_STATUS}
}

UPGRADE_GOLANG_ONLY=0
UPGRADE_GOLANG_ALL_DEPENDENCIES=0
UPGRADE_SUBMODULES_ONLY=0

while getopts "aghs-:" optchar; do
	case "${optchar}" in
		-)
			case "${OPTARG}" in
				golang-all)
					UPGRADE_GOLANG_ALL_DEPENDENCIES=1
					;;
				golang-only)
					UPGRADE_GOLANG_ONLY=1
					;;
				help)
					usage 0
					;;
				submodules-only)
					UPGRADE_SUBMODULES_ONLY=1
					;;
				*)
					echo "ERROR: Unknown option --${OPTARG}" >&2
					echo ""
					usage 1
					;;
			esac
			;;
		a)
			UPGRADE_GOLANG_ALL_DEPENDENCIES=1
			;;
		g)
			UPGRADE_GOLANG_ONLY=1
			;;
		h)
			usage 0
			;;
		s)
			UPGRADE_SUBMODULES_ONLY=1
			;;
		*)
			echo "ERROR: Unknown option -${OPTARG}" >&2
			echo ""
			usage 1
			;;
	esac
done

if [ $UPGRADE_GOLANG_ONLY -eq 1 -a $UPGRADE_SUBMODULES_ONLY -eq 1 ]; then
	echo "ERROR: Cannot combine -g (--golang-only) and -s (--submodules-only) options " >&2
	usage 1
fi

if [ $UPGRADE_GOLANG_ALL_DEPENDENCIES -eq 1 -a $UPGRADE_SUBMODULES_ONLY -eq 1 ]; then
	echo "ERROR: Cannot combine -a (--golang-all) and -s (--submodules-only) options " >&2
	usage 1
fi

readonly SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

if [ $UPGRADE_SUBMODULES_ONLY -ne 1 ]; then
	readonly UPGRADE_MODULES_SH=$SCRIPT_DIR/upgrade-go-modules.sh
	if [ $UPGRADE_GOLANG_ALL_DEPENDENCIES -eq 1 ]; then
		"$UPGRADE_MODULES_SH" --all
	else
		"$UPGRADE_MODULES_SH"
	fi
fi


if [ $UPGRADE_GOLANG_ONLY -ne 1 ]; then
	readonly REPOSITORY_DIR=$(cd "${SCRIPT_DIR}/../.." &> /dev/null && pwd)
	cd "${REPOSITORY_DIR}"
	git submodule update --remote
fi
