## Run migrations

Run migrations from the root dir:

```sh
go run ./internal/scripts/upmigration/main.go --dsn='file:main.sqlite?cache=shared&mode=rwc' --path="$(pwd)/migrations"
```

Alternatively set a `$ROOT_DIR` env var and use

```sh
go run ./internal/scripts/upmigration/main.go --dsn='file:main.sqlite?cache=shared&mode=rwc' --path="$ROOT_DIR/migrations"
```

## Generate SQL builder and model types

```sh
jet -source='sqlite' -dsn="file:main.sqlite" -path=./internal/.gen
```

## Clean start

Chain all the things

```sh
rm main.sqlite && touch main.sqlite && go run ./internal/scripts/upmigration/main.go --dsn='file:main.sqlite?cache=shared&mode=rwc' --path="$(pwd)/migrations" && jet -source='sqlite' -dsn="file:main.sqlite" -path=./internal/.gen
```
