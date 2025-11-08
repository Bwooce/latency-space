# SOCKS5 Port Assignments

## Port-per-Body Design

Each celestial body runs its own SOCKS5 proxy on a dedicated port. This solves the hostname routing limitation where SOCKS5 protocol cannot determine which proxy hostname the client used after DNS resolution.

## Port Assignment Table

### Tier 1: Planets (Most Common Use Cases)

| Port | Body | Type | Typical Latency | Priority |
|------|------|------|-----------------|----------|
| **1080** | **Mars** | Planet | 3-22 minutes | ⭐ PRIMARY (standard SOCKS5 port) |
| 1081 | Moon | Satellite | 1.3 seconds | ⭐ Fast testing |
| 1082 | Venus | Planet | 2-14 minutes | High |
| 1083 | Mercury | Planet | 4-12 minutes | High |
| 1084 | Jupiter | Planet | 32-54 minutes | High |
| 1085 | Saturn | Planet | 68-84 minutes | Medium |
| 1086 | Uranus | Planet | 2.5-3.2 hours | Medium |
| 1087 | Neptune | Planet | 4.0-4.7 hours | Medium |

### Tier 2: Dwarf Planets & Outer Solar System

| Port | Body | Type | Typical Latency | Priority |
|------|------|------|-----------------|----------|
| 1090 | Pluto | Dwarf Planet | 4.5-7 hours | Low |
| 1091 | Ceres | Dwarf Planet | 14-33 minutes | Low |
| 1092 | Eris | Dwarf Planet | 10-14 hours | Low |
| 1093 | Haumea | Dwarf Planet | 6-9 hours | Low |
| 1094 | Makemake | Dwarf Planet | 6-9 hours | Low |

### Tier 3: Moons (Interesting Targets)

| Port | Body | Type | Parent | Typical Latency | Priority |
|------|------|------|--------|-----------------|----------|
| 2080 | Io | Moon | Jupiter | 32-54 minutes | Medium |
| 2081 | Europa | Moon | Jupiter | 32-54 minutes | High (popular) |
| 2082 | Ganymede | Moon | Jupiter | 32-54 minutes | Medium |
| 2083 | Callisto | Moon | Jupiter | 32-54 minutes | Medium |
| 2084 | Titan | Moon | Saturn | 68-84 minutes | High (popular) |
| 2085 | Enceladus | Moon | Saturn | 68-84 minutes | Medium |
| 2086 | Triton | Moon | Neptune | 4.0-4.7 hours | Low |
| 2087 | Phobos | Moon | Mars | 3-22 minutes | Low |
| 2088 | Deimos | Moon | Mars | 3-22 minutes | Low |

### Tier 4: Spacecraft (Special Interest)

| Port | Body | Type | Typical Distance | Typical Latency | Priority |
|------|------|------|------------------|-----------------|----------|
| 3080 | Voyager 1 | Spacecraft | ~24 billion km | 22+ hours | High (iconic) |
| 3081 | Voyager 2 | Spacecraft | ~20 billion km | 18+ hours | Medium |
| 3082 | New Horizons | Spacecraft | ~7.5 billion km | 7 hours | Medium |
| 3083 | Parker Solar Probe | Spacecraft | Variable | 2-13 minutes | Medium |
| 3084 | JWST | Spacecraft | 1.5 million km | ~5 seconds | High (fast) |
| 3085 | Mars Perseverance | Spacecraft | On Mars | 3-22 minutes | Medium |

## Implementation Notes

### Why Mars Gets Port 1080?
- 1080 is the standard/default SOCKS5 port
- Mars is the most iconic/popular destination for latency simulation
- Makes the most common use case easiest: `--socks5-hostname latency.space:1080`

### Port Ranges
- **1080-1089**: Main planets (most frequently used)
- **1090-1099**: Dwarf planets and outer objects
- **2080-2099**: Major moons
- **3080-3099**: Spacecraft and special objects

### Docker Compose Strategy
For initial deployment, implement Tier 1 (planets) + popular bodies:
- Mars, Moon, Venus, Mercury, Jupiter, Saturn (ports 1080-1085)
- Europa, Titan (ports 2081, 2084)
- Voyager 1, JWST (ports 3080, 3084)

**Total for initial deployment: ~12 containers**

Can expand to all bodies later based on usage/demand.

## User Documentation

### Quick Reference
```bash
# Mars (most common - standard port)
curl --socks5-hostname latency.space:1080 https://example.com

# Moon (fastest - great for testing)
curl --socks5-hostname latency.space:1081 https://example.com

# Jupiter (outer planet)
curl --socks5-hostname latency.space:1084 https://example.com

# Europa (Jupiter's moon)
curl --socks5-hostname latency.space:2081 https://example.com

# Voyager 1 (deepest space)
curl --socks5-hostname latency.space:3080 https://example.com
```

### Browser Configuration
For browsers, configure SOCKS5 proxy:
- **Host**: `latency.space`
- **Port**: Choose from table above (1080 for Mars, 1081 for Moon, etc.)
- **SOCKS version**: v5
- **Remote DNS**: Enabled (important!)

## Why Port-Per-Body?

**Technical Limitation**: After DNS resolution (`mars.latency.space` → `168.119.226.143`), the SOCKS5 protocol has no way to preserve which hostname the client originally used. The server only sees:
- Source IP: `<client-ip>`
- Destination IP: `168.119.226.143`
- Destination Port: `1080`

The string "mars.latency.space" is lost after DNS lookup.

**Solution**: Each celestial body gets its own port, allowing the server to route correctly based on the port number alone.

## Future Considerations

### Load Balancing
If a specific body becomes very popular, can run multiple instances:
```
Mars-1: 1080
Mars-2: 1180
Mars-3: 1280
```

### Port Discovery API
Could provide an API endpoint:
```bash
curl https://latency.space/api/ports
# Returns: {"mars": 1080, "moon": 1081, ...}
```

### Docker Scaling
Use docker-compose scale or Kubernetes to spin up/down proxy instances based on demand.
