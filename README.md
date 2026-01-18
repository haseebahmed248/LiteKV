# LiteKV

Redis-compatible in-memory database built from scratch in Go — no external database libraries.

## Goals

- Implement RESP protocol (compatible with redis-cli)
- Support core commands: GET, SET, DEL, EXPIRE, TTL
- Multiple data structures: Strings, Lists, Hashes, Sets
- TTL expiration with background cleanup
- Persistence (RDB snapshots)
- Pub/Sub messaging

## Status

🚧 Work in Progress

## Architecture

```
Client (redis-cli)
      |
      v (RESP protocol)
+-------------+
|   LiteKV    |
+-------------+
      |
+-----+-----+
|  Store    | <-- In-memory data
+-----------+
```

## Run

```bash
go run cmd/litekv/main.go
```

## Connect

```bash
redis-cli -p 6379
> SET foo bar
OK
> GET foo
"bar"
```

## Author

Haseeb Ahmed - [GitHub](https://github.com/haseebahmed248)
