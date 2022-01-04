#!/usr/bin/env python3

#
# Json tags of struct fields in .go files are expected to be in format:
#   - camel case
#   - abbreviations and acronyms that are not at the beginning of the word are capitalized (eg. clientID, eventsURL; but caPool)
#   - generally, the tag should match the name of the field
#
# Additionally, some structs are used in network communication have defined schemas in swagger.yaml files. This goes for:
#   cloud2cloud-connector/swagger.yaml defines LinkedCloud schema
#   http-gateway/swagger.yaml defines several components
#
# This script verifies that:
# 1) tags names begin with a lower-case letter and contain only expected (a-zA-Z0-9_) characters.
# 2) components with defined schemas in selected swagger files must have existing fields in Go structs with matching
#    protobuf/json tags.
#

import argparse
import fnmatch
import os
import re
import sys
import yaml

SCRIPT_DIRECTORY = os.path.dirname(os.path.realpath(__file__))
ROOT_DIRECTORY = os.path.realpath(SCRIPT_DIRECTORY + "/../..")

DESCRIPTION = 'Validate json tags in Go files'

parser = argparse.ArgumentParser(description=DESCRIPTION)
parser.add_argument('-v', '--verbose', help="print verbose output",  action='store_true')
parser.add_argument('-f', '--fields', help="print warnings if field names do not match json tags",  action='store_true')
args = parser.parse_args()

PROTO_NAME_PATTERN = re.compile("protobuf:\".*name=([^\",]+)[^\"]*\"")
PROTO_JSON_PATTERN = re.compile("protobuf:\".*json=([^\",]+)[^\"]*\"")
JSON_PATTERN = re.compile("json:\"([^\"]+)\"")

def fill_protojson_tag_from_str(proto_tags, line, file):
  """Extract json name from protobuf field annotation."""
  match = re.search(PROTO_JSON_PATTERN, line)
  if not match:
    # json field inside protobuf tag exists only if the json name is different form name
    match = re.search(PROTO_NAME_PATTERN, line)
  if not match:
    return line
  line = line.replace(match.group(), "")
  proto_tags.setdefault(file, []).append(match.group(1))
  return line

def fill_json_tag_from_str(json_tags, line, file):
  """Extract json tag from field annotation."""
  match = re.search(JSON_PATTERN, line)
  if not match:
    return ""
  json_tag = match.group(1).split(",")[0]
  if not json_tag or json_tag == "-":
    return ""
  json_tags.setdefault(file, []).append(json_tag)
  return json_tag

def fill_proto_and_json_tags_from_file(proto_tags, json_tags, file):
  """Extract all protobuf and json tags from given file and save them as list in dictionary."""
  first_warning = True
  with open(file, "r") as f:
    for line in f:
      line = line.strip()
      if line.startswith("//"):
        continue
      if 'protobuf:"' in line:
        line = fill_protojson_tag_from_str(proto_tags, line, file)
        continue

      if not 'json:' in line:
        continue
      json_tag = fill_json_tag_from_str(json_tags, line, file)

      if not json_tag or not args.fields:
        continue
      field_name = line.split(" ", 1)[0]
      field_name_cannonical = field_name.replace('_', '').lower()
      json_tag = json_tag.replace('_', '')
      json_tag_cannonical = json_tag.replace('_', '').lower()
      if field_name_cannonical != json_tag_cannonical:
        if first_warning:
          first_warning = False
          print("file: {}".format(file))
        print("\tWARNING: field name '{}' does not match json tag '{}'".format(field_name, json_tag))

def get_all_proto_and_json_tags():
  """Find all protobuf and json tags in .go files from given directory."""
  proto_tags = {}
  json_tags = {}
  exclude_dirs = set(["bundle", "dependency", "charts"])
  for root, dirnames, filenames in os.walk(ROOT_DIRECTORY, topdown=True):
    dirnames[:] = [d for d in dirnames if d not in exclude_dirs]
    for filename in fnmatch.filter(filenames, "*.go"):
      file = root + "/" + filename
      fill_proto_and_json_tags_from_file(proto_tags, json_tags, file)

  return proto_tags, json_tags

def validate_key_existence(key, tags):
  """Check that key with given name exists among tags."""
  found = False
  for file, tags in tags.items():
    if key in tags:
      if args.verbose:
        print("key {} found in {}".format(key, file))
      found = True

  if not found:
    print("ERROR: key {} does not exist in any Go file".format(key), file=sys.stderr)
  return found

def validate_component(component, tags):
  """Check that all properties from given swagger component have existing tag counterparts."""

  if not "properties" in component:
    return True

  exists = True
  for k, v in component["properties"].items():
    if args.verbose:
      print("{}:{}".format(k, v))
    exists = validate_key_existence(str(k), tags) and exists
    if not "type" in v:
      continue
    if v["type"] == "object":
      exists = validate_component(v, tags) and exists
    if v["type"] == "array" and "type" in v["items"] and v["items"]["type"] == "object":
      exists = validate_component(v["items"], tags) and exists

  return exists

def validate_component_from_file(file, component_name, tags):
  """Extract component with given name from swagger file and validate it."""
  with open(file, "r") as f:
    try:
      if args.verbose:
        print("{}".format(file))
      data = yaml.safe_load(f)
      schemas = data["components"]["schemas"]
      if not schemas:
        print("ERROR: schemas object not found")
        return False

      component = schemas[component_name]
      if not component:
        print("ERROR: {} schema not found".format(component_name))
        return False
      return validate_component(component, tags)
    except yaml.YAMLError as exc:
      print(exc)

    return False

def validate_components_from_file(file, excluded_components, tags):
  """Extract all components swagger file and validate each."""
  valid = True
  with open(file, "r") as f:
    try:
      if args.verbose:
        print("{}".format(file))
      data = yaml.safe_load(f)
      schemas = data["components"]["schemas"]
      if not schemas:
        print("ERROR: schemas object not found")
        return False
      for component_name, component in schemas.items():
        if args.verbose:
          print("component {}".format(component_name))
        if component_name in excluded_components:
          continue
        if not "type" in component or not component["type"] == "object":
          continue
        valid = validate_component(component, tags) and valid
    except yaml.YAMLError as exc:
      print(exc)

  return valid

def validate_tag_format(tag):
  """Check that tag begins with lower-case letter and contains only supported characters."""
  valid = True
  # must be alphanumeric
  if not re.match("^[a-z][a-zA-Z0-9_]*$", tag):
    valid = False

  if args.verbose:
    print("tag {}: {}".format(tag, valid))

  if not valid:
    print("ERROR: invalid tag {}".format(tag), file=sys.stderr)
  return valid

def validate_c2c_connector_swagger(json_tags):
  """Validate LinkedCloud schema from cloud2cloud-connector/swagger.yaml."""
  tags = {}
  for k, v in json_tags.items():
    if "/cloud2cloud-connector/" in k or "pkg" in k:
      tags[k] = v
  return validate_component_from_file(ROOT_DIRECTORY + "/cloud2cloud-connector/swagger.yaml", "LinkedCloud", tags)

def validate_http_gateway_swagger(proto_tags):
  """Validate schemas from http-gateway/swagger.yaml."""
  tags = {}
  for k, v in proto_tags.items():
    if "/grpc-gateway/" in k or "/resource-aggregate/" in k:
      tags[k] = v

  excluded_components = ["Error", "ResourceCreateContent", "ResourceCreatedContent"]
  return validate_components_from_file(ROOT_DIRECTORY + "/http-gateway/swagger.yaml", excluded_components, tags)

def find_and_validate_json_fields():
  """Find all protobuf and json tags in Go files from given directory and validate them."""
  proto_tags, json_tags = get_all_proto_and_json_tags()

  valid = True
  for tags in json_tags.values():
    for tag in tags:
      valid = validate_tag_format(tag) and valid

  valid = validate_c2c_connector_swagger(json_tags) and valid
  valid = validate_http_gateway_swagger(proto_tags) and valid

  return valid

if __name__ == "__main__":
  find_and_validate_json_fields() or sys.exit(1)
