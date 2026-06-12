# networkd-prefix-watcher

IPv6プレフィクスの変更を検知するツール

## DHCPv6-PDのばあい

`/64` 未満のプレフィクス長のみ対応しています

## RAの場合

インターフェースに割り当てられたアドレスをもとにプレフィクスを計算します

## How to use

```
Watch IPv6 prefix changes and trigger a systemd target

Usage:
  networkd-prefix-watcher [flags]

Flags:
      --debounce duration   debounce duration for netlink events (default 2s)
      --env-file string     environment file path (default "/run/networkd-prefix-watcher/prefix.env")
  -h, --help                help for networkd-prefix-watcher
  -i, --interface string    network interface to watch
      --mode string         detection mode: pd or ra
      --once                check once and exit
      --prefix-len int      IPv6 prefix length to watch (default -1)
      --state-file string   state file path (default "/run/networkd-prefix-watcher/state.json")
      --target string       systemd target to restart (default "networkd-prefix-changed.target")
      --trigger-on-start    trigger when no previous state exists
```

## License

Copyright (c) 2026 Rokoucha

Released under the MIT license, see LICENSE.
