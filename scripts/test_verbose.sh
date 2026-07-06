#!/usr/bin/env bash
set -u

GO_STATUS=0
FE_STATUS=0

printf '\n==> Go tests (verbose)\n'
go test -v ./... || GO_STATUS=$?

printf '\n==> Frontend tests (Node test runner, verbose spec reporter)\n'
node --test --test-reporter spec frontend-tests/*.test.mjs || FE_STATUS=$?

printf '\n==> Summary\n'
if [[ $GO_STATUS -eq 0 ]]; then
  printf 'Go tests: PASS\n'
else
  printf 'Go tests: FAIL (exit %d)\n' "$GO_STATUS"
fi

if [[ $FE_STATUS -eq 0 ]]; then
  printf 'Frontend tests: PASS\n'
else
  printf 'Frontend tests: FAIL (exit %d)\n' "$FE_STATUS"
fi

if [[ $GO_STATUS -eq 0 && $FE_STATUS -eq 0 ]]; then
  printf 'Overall: PASS\n'
  exit 0
fi

printf 'Overall: FAIL\n'
exit 1
