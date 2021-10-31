
This is a fork of [github.com/go-gorm/sqlite](github.com/go-gorm/sqlite)

That works with [modernc.org/sqlite](modernc.org/sqlite) which is a pure-go sqlite
implementation. Obviously, because `modernc.org/sqlite` is a re-implementation of sqlite 
there might be missing features and stability issues. It should work for development or simple use-cases.

# GORM Sqlite Driver

![CI](https://github.com/cloudquery/sqlite/workflows/CI/badge.svg)

## USAGE

```go
import (
  "gorm.io/ssbeatty/sqlite"
  "gorm.io/gorm"
)

// gitlab.com/modernc.org/sqlite
db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
```

Checkout [https://gorm.io](https://gorm.io) for details.
