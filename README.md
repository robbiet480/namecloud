# namecloud
A little utility to point all your Namecheap domains to Cloudflare and optionally, transfer them.

# BROKEN

This is broken while waiting for [this `go-namecheap` PR](https://github.com/billputer/go-namecheap/pull/28) and [this `cloudflare-go` PR](https://github.com/cloudflare/cloudflare-go/pull/491) to be merged.

# Usage

All flags are required.

```
namecloud is a little utility to point all your Namecheap domains to Cloudflare nameservers and optionally, transfer them.

Usage:
  namecloud [command]

Available Commands:
  help        Help about any command
  point       point will configure your Namecheap domain(s) for use with Cloudflare
  transfer    transfer will complete most of the process of transferring a domain to Cloudflare Registrar.

Flags:
      --cloudflare.account-id string   Cloudflare Account ID
      --cloudflare.api-key string      Cloudflare API Key
      --cloudflare.email string        Cloudflare Email
  -h, --help                           help for namecloud
      --namecheap.api-token string     Namecheap API Token
      --namecheap.api-user string      Namecheap API User
      --namecheap.username string      Namecheap Username

Use "namecloud [command] --help" for more information about a command.
```

# LICENSE

MIT
