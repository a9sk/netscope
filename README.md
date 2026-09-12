# netscope

A tiny terminal network monitor. Watch live connections, packet flows, and bandwidth sparklines.

## Flags

- `-i <interface>` — network interface to capture and monitor (default: `any`)
- `-p <port>` — port filter; accepts a bare number (`80`) or a tshark filter expression (`tcp port 443`)

Example:

```bash
go run . -i eth0 -p 443
go run . -i wlan0 -p "udp port 53"
```

## Controls

- `↑/↓` or `k/j` — scroll
- `pgup` / `pgdown` — page scroll
- `home` / `g` — jump to top
- `end` / `G` — jump to bottom
- `c` — toggle connections / packet capture view
- `/` — live-as-you-type filter (matches protocol, addresses, ports, process name, PID, or length)
- `p` — pause/unpause the display; capture and stats keep running in the background
- `s` — cycle sort column for the current view
- `d` — open/close detail panel for the selected item (replaces the footer)
- `e` — start/stop exporting captured packets to `netscope.pcap`
- `q` / `esc` / `ctrl+c` — quit

## PCAP export

Press `e` to start writing packets to `netscope.pcap` in the current directory.
The export uses the same interface and port filter passed on the command line.
Press `e` again to stop the capture. The filename is always `netscope.pcap`.
