#!/usr/bin/env bash
# This script will check all subaccounts in the current KCP environment. It executes the "edp get" command for each subaccount.
# Before running this script, make sure you have the KCPCONFIG set to the correct value.
# Make sure you have the edp environment variables set correctly (see README.md)


subaccounts=$(kcp rt -o custom=Subaccount:subAccountID | sort | uniq)

for subaccount in $subaccounts; do
    echo "Checking subaccount $subaccount"
    ./edp get $subaccount
done

