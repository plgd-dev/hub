#!/usr/bin/env python3

#
# Download and install latest prebuild protoc
#

import argparse
import json
import os
import re
import shutil
import subprocess
import stat
import sys

DESCRIPTION = 'Download and install latest protoc'

parser = argparse.ArgumentParser(description=DESCRIPTION)
parser.add_argument('-v', '--version', help="show the latest available protoc version (no download)",
  action='store_true')
args = parser.parse_args()

# get path to latest version from github repository
PROTOC_REPOSITORY = 'https://api.github.com/repos/protocolbuffers/protobuf/releases/latest'
result = subprocess.run(['curl', '--silent', PROTOC_REPOSITORY],
  stdout=subprocess.PIPE)

packages = json.loads(result.stdout.decode('utf-8'))

if args.version:
  print(packages['tag_name'])
  sys.exit(0)

download_path = ''
for package in packages['assets']:
  if re.search('linux.*x86_64', package['name']):
    download_path = package['browser_download_url']
    break

if not download_path:
  print("ERROR: failed to obtain download path", file=sys.stderr)
  sys.exit(1)

SCRIPT_DIRECTORY = os.path.dirname(os.path.realpath(__file__))
os.chdir(SCRIPT_DIRECTORY)

# download
subprocess.run(['curl', '-OL', download_path])
downloaded_file = os.path.basename(download_path)

# unzip
PROTOC_DIRECTORY = SCRIPT_DIRECTORY + "/protoc"
shutil.unpack_archive(downloaded_file, PROTOC_DIRECTORY)

def copy_files_to_directory(src_dir, out_dir, set_executable = False):
  if not os.path.exists(out_dir):
    os.makedirs(out_dir)
    os.chmod(out_dir, stat.S_IRWXU | stat.S_IRGRP | stat.S_IXGRP | stat.S_IROTH | stat.S_IXOTH)

  files = os.listdir(src_dir)
  for file in files:
    src_file = os.path.join(src_dir, file)
    if os.path.isdir(src_file):
      subprocess.run(['cp', '-rp', src_file, out_dir])
    else:
      shutil.copy(src_file, out_dir)
    if set_executable:
      out_file = os.path.join(out_dir, file)
      os.chmod(out_file, stat.S_IRWXU | stat.S_IRGRP | stat.S_IXGRP | stat.S_IROTH | stat.S_IXOTH)

# move files to /usr/local
PROTOC_BIN_DIR = PROTOC_DIRECTORY + "/bin"
BIN_DIR = "/usr/local/bin"
copy_files_to_directory(PROTOC_BIN_DIR, BIN_DIR, True)

PROTOC_INCLUDE_DIR = PROTOC_DIRECTORY + "/include"
INCLUDE_DIR = "/usr/local/include"
copy_files_to_directory(PROTOC_INCLUDE_DIR, INCLUDE_DIR)

# clean-up
shutil.rmtree(PROTOC_DIRECTORY)
os.remove(downloaded_file)
