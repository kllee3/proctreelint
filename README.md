# proctreelint

A linter for process-tree snapshots. Point it at a text dump of pids,
their parent pids, and their state, and it reports structural problems
with line numbers: pids that claim a parent that's never defined,
duplicate pids, processes that are their own parent, zombies that still
have children attached.

The kind of input this is meant for: a `ps -eo pid,ppid,state,comm`
dump, a flattened walk of `/proc`, or logs from something that tracks
fork/exit events across a fleet of hosts. Those can get big -
concatenate a day of snapshots across a few hundred containers and
you're well past what you want sitting in memory as one buffer. So
`Lint` reads line by line with `bufio.Scanner` and never reads the
whole input into memory at once. The only state it keeps around is one
small record per pid (pid, ppid, line number, state) - proportional to
the number of processes, not to the size of the input.

## Input format

One process per line, whitespace-separated fields:

```
pid ppid state comm
```

`comm` may contain spaces and takes the rest of the line. Blank lines
and lines starting with `#` are ignored. A `ppid` of `0` marks a root
and is never flagged as missing.

## Usage

```
go run . examples/sample.pt
```

`examples/sample.pt` has a few problems planted in it:

```
# pid   ppid  state  comm
1       0     S      init
100     1     S      sshd
205     100   S      bash
340     205   S      stress
501     340   Z      stress-child
610     501   S      grandchild-of-zombie
9012    9012  S      loopy
7777    4444  S      orphan-ish
205     100   S      bash
```

Running the linter on it prints:

```
examples/sample.pt:5: zombie-with-children: pid 501 is a zombie (state Z) but still has children reparented to it
examples/sample.pt:7: self-parent: pid 9012 lists itself as its own parent
examples/sample.pt:8: unknown-parent: pid 7777 has ppid 4444, which never appears in the input
examples/sample.pt:9: duplicate-pid: pid 205 already defined on line 3
```

With no findings the command exits `0` and prints nothing. With
findings it exits `1`. A malformed line (too few fields, a pid that
doesn't parse as an integer) is reported as `parse-error` rather than
aborting the run, so one bad line in a large snapshot doesn't hide the
rest of the findings.

Pass a filename, or `-` (or nothing) to read from stdin:

```
ps -eo pid,ppid,state,comm | tail -n +2 | go run . -
```

## Rules

| rule                   | flags                                                          |
|-------------------------|-----------------------------------------------------------------|
| `parse-error`            | a line with too few fields or a non-numeric pid/ppid            |
| `self-parent`            | a pid that lists itself as its own ppid                        |
| `duplicate-pid`          | the same pid defined more than once                             |
| `unknown-parent`         | a ppid (other than 0) that no line in the input ever defines    |
| `zombie-with-children`   | a process in state `Z` that still has children pointing at it   |
| `parent-cycle`           | a ppid chain that loops back on itself over two or more hops    |

## What this doesn't do yet

It doesn't know anything about real `/proc` semantics beyond what's
in the four columns above; it lints the snapshot as given, not the
live system.

## License

MIT, see [LICENSE](LICENSE).
