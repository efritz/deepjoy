# Deepjoy

[![GoDoc](https://godoc.org/github.com/efritz/deepjoy?status.svg)](https://godoc.org/github.com/efritz/deepjoy)
[![Build Status](https://secure.travis-ci.org/efritz/deepjoy.png)](http://travis-ci.org/efritz/deepjoy)
[![Maintainability](https://api.codeclimate.com/v1/badges/087a251bad276ff18318/maintainability)](https://codeclimate.com/github/efritz/deepjoy/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/087a251bad276ff18318/test_coverage)](https://codeclimate.com/github/efritz/deepjoy/test_coverage)

A Go library that wraps [gomodule/redigo](https://github.com/gomodule/redigo)
in a pool. This library is named after [Y-40](http://www.y-40.com/en/).

## Example

First, instantiate a client with any of hte following optional config functions.
The only required parameter is the address of the remote Redis server - all other
values have sane (hopefully) defaults.

```go
client := NewClient(
    "dart.it.corp:6379",
    WithPassword("hunter2"),
    WithDatabase(3),
    WithConnectTimeout(time.Second * 5),
    WithReadTimeout(time.Second * 5),
    WithWriteTimeout(time.Second * 5),
    WithBorrowTimeout(time.Second * 5),
    WithPoolCapacity(10),
    WithBreaker(overcurrent.NewCircuitBreaker()),
    WithLogger(NewLogAdapter()),
)

// Drain the pool
defer client.Close()
```

Password, database, connect, read, and write timeouts are sent directly to the
backing redis library. The borrow timeout setting is used to determine how much
time we will spend waiting on an *empty* pool before returning a no connection
error back to the user.

The breaker is an instance of an [overcurrent](https://github.com/efritz/overcurrent)
circuit breaker and is invoked when dialing a new redis connection. If dials are
failing very rapidly, it is best to back off on the consumer side to let the remote
service recover.

A logger is a simple interface with a single Printf function. By default, the
logger will call `log.Printf`. `NewNilLogger` will return a silent logger. If
your application uses a more sophisticated (structured) logging library, a small
shim can be created so that it conforms to this protocol.

The client API is otherwise minimal. You can run a redis command, which consists of
a single string command and a following variadic list of interfaces composing the
command's arguments as follows.

```go
result, err := client.Do("hmset", "myhash", "f1", "Hello", "f2", 14)
if err != nil {
    // handle error
}

// parse interface{} result
```

In order to run multiple commands in a single round trip, you can use a pipeline.
Commands added to the pipeline do not have an effect on the remote server - no
network communication is done until the pipeline is run. Piplines are NOT atomic
or transactional - it is possible for command in the pipeline to fail and for
side-effects from earlier commands to persist. THe result from a pipeline is a
slice of interfaces - the result of each command is returned in the order that it
was added to the pipeline.

```go
pipeline := client.Pipeline()
pipeline.Add("GET", "foo")
pipeline.Add("GET", "bar")
pipeline.Add("GET", "baz")

results, err := pipeline.Run()
if err != nil {
    // handle error
}

// parse interface{} result
```

## License

Copyright (c) 2017 Eric Fritz

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
