#### How to deploy

```
export AWS_ACCESS_KEY_ID=?
export AWS_SECRET_ACCESS_KEY=?
```

```
GOOS=linux GOARCH=amd64 go build ./example/main.go && cp main example 
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