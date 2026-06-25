# C-recovery pop-slice reduction version retention

Date: 2026-06-25

## Forward repair

Reviewer blockers in the same-pop-target reduction path were real:

- same-pop collapse picked a survivor using whole-stack cost and stack score,
  while upstream C selects the temporary parent subtree by error cost, dynamic
  precedence, equal-positive-error replacement, then recursive subtree order;
- action-local same-pop alternatives could be generically merged into older
  header-equivalent versions before the action had collapsed all of its own
  sibling slices;
- trailing extras must not define the same-pop child-selection group. C removes
  them, selects the parent children, then replays the selected extras.

The repair stays grammar-neutral:

- `reduceForkWindowPreference` now follows `ts_parser__select_tree` ordering as
  closely as the Go raw `Node` model supports;
- same-pop fork grouping is by original pop target and top state, not by
  trailing-extra equality;
- `cDoAllPotentialReductions` collects all candidates from one reduce action,
  collapses same-pop candidates before generic header-equivalence merge, and
  records `STACK_VERSION_NONE` when an action only merges into older versions;
- same-pop collapse walks past replayed trailing extras to find the reduced
  parent and original pop target.

Focused coverage added or updated:

- same-pop C-recovery reduction asserts the surviving parent child array;
- same-pop C-recovery with trailing extras asserts the selected parent and
  replayed selected extra;
- action-local same-pop collapse-before-older-merge asserts the older merged
  link receives only the selected parent;
- reducer selection tests now expect C-style recursive-order coalescing for
  raw-only same-pop alternatives.

## Verification

Host:

```text
go test . -run 'TestCDoAllPotentialReductions|TestCCollectPotentialReductions' -count=1
ok  	github.com/odvcencio/gotreesitter	0.016s

go test . -run 'TestSelectReduceForkChildren|TestFaithfulGSSMergeRecursesPredecessorLinksAndReduceSelectsConstructedParent|TestReduceForkSelection|TestFaithfulForkReduce' -count=1
ok  	github.com/odvcencio/gotreesitter	0.032s
```

Requested broader host check:

```text
go test . -run 'TestC' -count=1
```

Result:

```text
--- FAIL: TestCompactFullLeafSizeBudget
    glr_test.go:22: compactFullLeaf size = 68, want 60
--- FAIL: TestCppMalformedClassFunctionDefinitionRecovery
    parser_result_cpp_recovery_test.go:58: root.HasError = false, want true
FAIL
```

The C++ failure also reproduces with `GOT_C_RECOVERY=0`, so it is not caused by
the C-recovery reduction changes. The compact leaf size failure is tied to
unrelated `glr.go` layout changes already present in the dirty worktree.

No Docker parity witness was rerun in this pass.
