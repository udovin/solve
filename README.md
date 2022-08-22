# Solve

[![GoDoc](https://godoc.org/github.com/udovin/solve?status.svg)](https://godoc.org/github.com/udovin/solve)
[![codecov](https://codecov.io/gh/udovin/solve/branch/master/graph/badge.svg)](https://codecov.io/gh/udovin/solve)
[![Go Report Card](https://goreportcard.com/badge/github.com/udovin/solve)](https://goreportcard.com/report/github.com/udovin/solve)

Solve is distributed under Apache 2.0 License.

# How to start development

First of all you should build `solve` binary:

```bash
go build .
```

Then create config file `config.json` with following contents:

```json
{
  "db": {
    "driver": "sqlite",
    "options": {
      "path": "database.sqlite"
    }
  },
  "server": {
    "port": 4242
  },
  "invoker": {
    "threads": 1
  },
  "security": {
    "password_salt": "qwerty123"
  },
  "storage": {
    "files_dir": ".data/files"
  },
  "log_level": "debug"
}
```

Then apply database migrations:

```bash
./solve migrate --create-data
```

Then run server (API will be available on `http://localhost:4242`):

```bash
./solve server
```

Then you can register new `admin` user with password `qwerty123`:

```bash
curl -XPOST \
    -F 'email=admin@gmail.com' \
    -F 'login=admin' \
    -F 'password=qwerty123' \
    'http://localhost:4242/api/v0/register'
```

After that you can grant this user admin permissions (`admin_group` role):

```bash
curl -XPOST \
    --unix-socket '/tmp/solve-server.sock' \
    's/socket/v0/users/admin/roles/admin_group'
```
