#!/usr/bin/env bash
find . -type f -print0 | xargs -0 perl -pi -e 's#github.com/kyma-project/kyma-environment-broker#github.com/kyma-project/kyma-environment-broker#g'