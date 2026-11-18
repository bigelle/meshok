# Usage (planned)

## CLI

### Check status:
```
meshok
```
This should check if the daemon is online and print devices that have requested pairing or some other useful info.

### Start daemon:
```sh
meshok serve
```
This should check if the daemon is already running, and run it if it's not.

### Stop daemon:
```sh
meshok shutdown
```
This should send an IPC call to the daemon requesting shutdown.

### List available devices:
```sh
meshok devices 
```
This should send an IPC call to the daemon requesting the list of found devices.

### Connect to the device:
```sh
meshok devices connect <device_id>
```
This should send an IPC call to the daemon asking to pair with <device_id>.

### Send a file:
```sh
meshok send <device_id> fileA.txt fileB.png fileC.mp4
```

### Accept a file:
```sh
meshok accept <device_id | request_id> 
```

## Everything past this point is weird, might not work and might change multiple of times.

### Sync directory:
```sh
meshok sync /path/to/dir
```

### Sync directory with only one connected device:
```sh
meshok sync --device=<device_id> /path/to/dir
```

### Accept sync request:
```sh
meshok sync --accept <device_id>

