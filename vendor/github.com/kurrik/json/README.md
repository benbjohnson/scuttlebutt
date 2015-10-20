json
====
This project implements a read-only JSON parser in Go

This file provides a custom JSON parser suitable for processing Twitter data.
Some differences from the standard Golang json package:
  * Parses numbers into int64 where possible, float64 otherwise.
  * Faster!  Probably due to less reflection:

Real-world Tweet example (Std is encoding/json):
<pre>
BenchmarkStdJSON          10000    158362 ns/op
BenchmarkJSON             10000    122690 ns/op
</pre>

The following numbers use a contrived small example.  Untyped unmarshals
into an `[]interface{}` while Bucket unmarshals into a type using reflection:

<pre>
BenchmarkStdBucket       200000     10781 ns/op
BenchmarkBucket          500000      5455 ns/op
BenchmarkUntypedBucket  1000000      2747 ns/op
</pre>

This library also performs escaped UTF-8 entity decoding on strings, so you
get the raw runes instead of escaped `\u####` sequences.

Maturity
--------
This is a relatively new JSON parser so there's certainly edge cases
which may throw errors or otherwise parse incorrectly.  Please file bugs /
submit tests!  I'm happy to fix issues.

That being said, this library is very compatible with Twitter's JSON encoding.
It has been able to parse messages from Twitter's 1% stream for hours without
issue.

Installing
----------
Run

    go get github.com/kurrik/json

Include in your source:

    import "github.com/kurrik/json"

Updating
--------
Run

    go get -u github.com/kurrik/json

Benchmarking
------------
<pre>
go test -bench=".*"
</pre>

Testing
-------
Most edge cases are tested in `compat_test.go`.  Please add cases if you find
any!
