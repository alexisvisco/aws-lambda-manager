```
Deploy on amazon, google cloud, azure in a way that is absolutely simple and efficient for you.

Usage:
  expected [flags]
  expected [command]

Available Commands:
  help        Help about any command
  lambda      Manage deployment for a lambda
    deploy       Create or update a lambda
    list         List of lambdas
    list-version List of version for a given lambda
    rollback     Rollback a lambda to a certain version

Flags:
  -h, --help            help for expected
      --region string   region to use (default "eu-west-3")

Use "expected [command] --help" for more information about a command.
```

#### How to deploy

```
export AWS_ACCESS_KEY_ID=?
export AWS_SECRET_ACCESS_KEY=?
```

```
cd repo-test && GOOS=linux GOARCH=amd64 go build -o main main.go && cd -
go run cmd/cli/main.go lambda deploy repo-test repo-test
```

#### How to update with new code

```
cd repo-test && sed -i 's/lol/ultra-cool/g' main.go && GOOS=linux GOARCH=amd64 go build -o main main.go && cd - 
go run cmd/cli/main.go lambda deploy repo-test repo-test --id $ID_GIVEN_BY_PREVIOUS_COMMAND
```

Now we can list different versions

```
go run cmd/cli/main.go lambda list-version repo-test $ID_GIVEN_BY_PREVIOUS_COMMAND
```

Output:
```
- time: 04 May 19 18:02 CEST    sha256: 3b042752b0fd
- time: 04 May 19 18:08 CEST    sha256: 483abd235caa
```

#### How to rollback

In my case sha256 can be 3b042752b0fd or 3b
```
go run cmd/cli/main.go lambda rollback repo-test $ID_GIVEN_BY_PREVIOUS_COMMAND $SHA256
```