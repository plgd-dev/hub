#!/usr/bin/env python3

#
# Keys in yaml files are expected to be in format:
#   - camel case
#   - abbreviations and acronyms that are not at the beginning of the word are capitalized (eg. clientID, eventsURL; but caPool)
#
# Additionally, keys in config.yaml files should have corresponding yaml tags in golang structs.
#
# This script verifies that
# 1) key names begin with a lower-case letter and contain only expected (a-zA-Z0-9_\-) characters.
# 2) a .go file with a field with the given yaml tag exists
#

import argparse
import fnmatch
import os
import re
import sys
import yaml

SCRIPT_DIRECTORY = os.path.dirname(os.path.realpath(__file__))
ROOT_DIRECTORY = os.path.realpath(SCRIPT_DIRECTORY + "/../..")

DESCRIPTION = 'Validate keys format in YAML files'

parser = argparse.ArgumentParser(description=DESCRIPTION)
parser.add_argument('-v', '--verbose', help="print verbose output",  action='store_true')
parser.add_argument('-f', '--fields', help="print warnings if field names do not match yaml tags",  action='store_true')
args = parser.parse_args()

YAML_PATTERN = re.compile("yaml:\"([^\"]+)\"")

def validate_yaml_key_format(key):
  """Check that key begins with lower-case letter and contains only supported characters."""
  valid = True
  # must be alphanumeric
  if not re.match("^[a-z][a-zA-Z0-9_]*$", key):
    valid = False

  if args.verbose:
    print("key {}: {}".format(key, valid))

  if not valid:
    print("ERROR: invalid key {}".format(key), file=sys.stderr)
  return valid

def validate_yaml_keys_format(data):
  """Recursively validate all keys in a dictionary."""
  valid = True
  for k, v in data.items():
    valid = validate_yaml_key_format(str(k)) and valid
    if isinstance(v, dict):
      valid = validate_yaml_keys_format(v) and valid
    elif isinstance(v, list):
      for item in v:
        if isinstance(item, dict):
          valid = validate_yaml_keys_format(item) and valid
  return valid

def fill_yaml_tags_from_file(go_tags, file):
  """Extract all yaml tags from given file and save them as list in dictionary."""
  with open(file, "r") as f:
    for line in f:
      if not 'yaml:' in line:
        continue
      line = line.strip()
      if line.startswith("//"):
        continue
      match = re.search(YAML_PATTERN, line)
      if not match:
        continue
      tag = match.group(1).split(",")[0]
      if not tag or tag == "-":
        continue
      go_tags.setdefault(file, []).append(tag)

      if not args.fields:
        continue
      name = line.split(" ", 1)[0]
      if name.lower() != tag.lower():
        print("WARNING: field name '{}' does not match yaml tag '{}'".format(name, tag))

def get_all_yaml_tags(dir = ROOT_DIRECTORY):
  """Find all yaml tags in .go files from given directory."""
  go_tags = {}
  exclude_dirs = set(["bundle", "dependency", "charts"])
  for root, dirnames, filenames in os.walk(dir, topdown=True):
    dirnames[:] = [d for d in dirnames if d not in exclude_dirs]
    for filename in fnmatch.filter(filenames, "*.go"):
      file = root + "/" + filename
      fill_yaml_tags_from_file(go_tags, file)

  return go_tags

def validate_yaml_key_existence(key, go_tags):
  """Check that for given yaml key from configuration file exists a corresponding yaml tag of some field in a Go struct."""
  for file, tags in go_tags.items():
    if key in tags:
      if args.verbose:
        print("key {} found in {}".format(key, file))
      return True

  print("ERROR: key {} does not exist in any .go file".format(key), file=sys.stderr)
  return False

def validate_yaml_keys_existence(data, go_tags):
  """Check that all fields from configuration yaml file have equivalent yaml tag in some Go file."""
  exists = True
  for k, v in data.items():
    exists = validate_yaml_key_existence(str(k), go_tags) and exists
    if isinstance(v, dict):
      exists = validate_yaml_keys_existence(v, go_tags) and exists
    elif isinstance(v, list):
      for item in v:
        if isinstance(item, dict):
          exists = validate_yaml_keys_existence(item, go_tags) and exists
  return exists

def find_and_validate_yaml_file(file, go_tags):
  """Validate format and existence of yaml keys from given file."""
  with open(file, "r") as f:
    try:
      if args.verbose:
        print("{}".format(file))
      data = yaml.safe_load(f)
      valid = validate_yaml_keys_format(data)
      filename = os.path.basename(file)
      if filename == "config.yaml":
        valid = validate_yaml_keys_existence(data, go_tags) and valid
    except yaml.YAMLError as exc:
      print(exc)
  return valid

def find_and_validate_yaml_files(dir = ROOT_DIRECTORY):
  """Find all yaml files in directory and validate them."""
  go_tags = get_all_yaml_tags()

  valid = True
  exclude_dirs = set(["dependency", "templates"])
  exclude_filenames = set(["swagger.yaml"])
  for root, dirnames, filenames in os.walk(dir, topdown=True):
    dirnames[:] = [d for d in dirnames if d not in exclude_dirs]
    for filename in fnmatch.filter(filenames, "*.yaml"):
      if filename in exclude_filenames:
        continue
      file = root + "/" + filename
      valid = find_and_validate_yaml_file(file, go_tags) and valid

  return valid

if __name__ == "__main__":
  find_and_validate_yaml_files() or sys.exit(1)
