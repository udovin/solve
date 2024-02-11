# Solve

[![GoDoc](https://godoc.org/github.com/udovin/solve?status.svg)](https://godoc.org/github.com/udovin/solve)
[![codecov](https://codecov.io/gh/udovin/solve/branch/master/graph/badge.svg)](https://codecov.io/gh/udovin/solve)
[![Go Report Card](https://goreportcard.com/badge/github.com/udovin/solve)](https://goreportcard.com/report/github.com/udovin/solve)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/6577/badge)](https://bestpractices.coreinfrastructure.org/projects/6577)

Solve is distributed under the terms of both the MIT license and the Apache 2.0 License.

# How to start development

First of all you should build `solve` and `safeexec` binaries:

```bash
make all
```

Then create config file `config.json` with following contents:

```json
{
  "db": {
    "driver": "sqlite",
    "options": {
      "path": "data/db.sqlite"
    }
  },
  "server": {
    "port": 4242
  },
  "invoker": {
    "workers": 1,
    "safeexec": {
      "path": "cmd/safeexec/safeexec"
    }
  },
  "security": {
    "password_salt": "qwerty123"
  },
  "storage": {
    "driver": "local",
    "options": {
      "files_dir": "data/files"
    }
  },
  "log_level": "debug"
}
```

Then apply database migrations:

```bash
mkdir -p data && ./cmd/solve/solve migrate && ./cmd/solve/solve migrate-data
```

Then run server (API will be available on `http://localhost:4242`):

```bash
./cmd/solve/solve server
```

Then you can register new `admin` user with password `qwerty123`:

```bash
./cmd/solve/solve client create-user \
  --login admin \
  --password qwerty123 \
  --email admin@gmail.com \
  --add-role admin_group
```
