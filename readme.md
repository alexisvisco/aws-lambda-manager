### Overview 

[![asciicast](https://asciinema.org/a/u2FoWPKvOOJzuaYuJqUx6TIlM.png)](https://asciinema.org/a/u2FoWPKvOOJzuaYuJqUx6TIlM)

### Commands

```go
Deploy on amazon in a way that is absolutely simple and efficient for you.
Included: 
 - Versions: using digest
 - Efficient storage: using s3 and zip your lambda
 - AWS Gateway setup

Usage:
  awsl [flags]
  awsl [command]

Available Commands:
  deploy       Create or update a lambda
  help         Help about any command
  list         List of lambdas
  list-version List of version for a given lambda
  remove       Remove a lambda
  rollback     Rollback a lambda to a certain version

Flags:
  -h, --help            help for awsl
      --region string   region to use (default "eu-west-3")

Use "awsl [command] --help" for more information about a command.
```