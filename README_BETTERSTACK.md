# BetterStack Beyla Fork

This is BetterStack's fork of [Grafana Beyla](https://github.com/grafana/beyla) with critical fixes for production use.

## Why This Fork?

This fork includes essential patches that fix memory leaks in Beyla when monitoring high-connection processes. The patches have been submitted upstream but are critical for production use.

## Patches Applied

### Memory Leak Fix
- **Issue**: Unbounded array growth when tracking processes with many ephemeral connections
- **Fix**: Deduplicates ports and caps the maximum tracked ports per process
- **Impact**: Prevents OOM kills when monitoring load generators or high-traffic clients
- **PR**: [Pending upstream submission]

## Docker Images

Pre-built Docker images with our patches are available at:
```
docker pull betterstack/beyla:latest
```

## Usage

This fork is a drop-in replacement for the official Beyla. Simply replace your image reference:

```yaml
# Before
image: grafana/beyla:latest

# After  
image: betterstack/beyla:latest
```

## Building From Source

```bash
# Clone this repository
git clone git@github.com:BetterStackHQ/beyla.git
cd beyla

# The patch is automatically applied during CI/CD
# To apply manually:
patch -p1 < fix-memory-leak-minimal.patch

# Build
make build
```

## Staying Up-to-Date

This fork is regularly synced with upstream Grafana Beyla. Our CI/CD automatically:
1. Applies our patches
2. Runs all tests
3. Builds multi-architecture Docker images
4. Pushes to DockerHub

## Contributing

Issues and PRs related to our patches should be opened in this fork. For general Beyla issues, please contribute upstream at [grafana/beyla](https://github.com/grafana/beyla).

## License

This fork maintains the same Apache 2.0 license as the original Beyla project.