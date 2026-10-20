# TaskFlow client source integration

This directory preserves the existing `pkg/cloudclient` implementation and tests
from the user's TaskFlow-Go development source. Its Go package remains
`cloudclient`. No sibling module `replace` or invented published version is needed
to build WeKnora.

Source snapshot: TaskFlow-Go local development branch based on
`ef1d68176f07dc9285d633f1f93549e6dbfb8b4a`, with its verification API additions.
The local vNext implementation is recorded at
`33d6cc31d80ce4c2607525b20baac43894945ac3`; it is a local implementation snapshot,
not the original repository history or a published SDK release.
The source is user-owned and currently carries no separate LICENSE. This copy does
not assert an MIT grant for it. Review licensing before third-party distribution.

To synchronize after an actual SDK change, explicitly provide its package path:

```text
python scripts/sync-taskflow-sdk.py <TaskFlow-source>/pkg/cloudclient
go test ./internal/integration/taskflowclient
```

The script copies the six named source/test files and directly compares their
bytes. Business orchestration lives in the Change Module, not in this client.
