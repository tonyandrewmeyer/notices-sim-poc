# Juju notices simulator

## Usage

The simplest usage is to just run:

```shell
./start.sh
```

(first modifying the script to have the right location for your Pebble). This
will start a tmux session that has Pebble, a dummy HTTP server, the simulator,
and a watch of the 'charm' log.

### Manual

In one console, navigate to the root Pebble directory, and start Pebble, e.g.:

```shell
mkdir -p /tmp/pebble && PEBBLE=/tmp/pebble go run ./cmd/pebble run
```

In another console, navigate to the root of this project, and start the simulator, e.g.:

```shell
go run sim.go
```

In another console, navigate to the root Pebble directory, and run Pebble commands, e.g.:

```shell
PEBBLE=/tmp/pebble go run ./cmd/pebble notify example.com/foo
PEBBLE=/tmp/pebble go run ./cmd/pebble exec echo 'Hello world!'
```

You should see in the simulator console logging entries like:

```
2024/06/26 19:43:10 INFO pebbleNoticer starting
2024/06/26 19:43:10 INFO sending Juju event type=change-update key=1 id=1
2024/06/26 19:43:10 INFO sending Juju event type=custom key=example.com/foo id=3
2024/06/26 19:43:10 INFO processing event in charm type=change-update key=1 eventID=1
2024/06/26 19:43:11 INFO processing event in charm type=custom key=example.com/foo eventID=3
```

Whenever the simulator would send an event to a charm, it runs `charm.py`
with the notice ID, notice key, and notice type as arguments. You can adjust
the Python code to do any sort of charm-like behaviour.

Note that the 'after' value for the notices query is not stored, so if you
restart the simulator without resetting the Pebble state, you'll get all
previous notices re-emitted.

### Services and checks

A [simple layer](base.yaml) is included to help with simulating check failures.
Add it like:

```shell
PEBBLE=/tmp/pebble go run ./cmd/pebble add base /path/to/sim/folder/base.yaml
```

If you want the checks to pass, start an HTTP server that listens on port 8080,
for example with:

```
python3 -m http.server 8080
```

You can also use it for service commands (e.g. start, stop, restart) - the
service is a simple `sleep` command, so doesn't really do anything.
