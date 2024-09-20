#!/bin/bash

cd "$(dirname "$0")/bin"

# remove all files from bin/<only digits> directories

find . -exec chmod u+w {} \;; ls|grep -E '^\d+$$'|xargs rm -rf
